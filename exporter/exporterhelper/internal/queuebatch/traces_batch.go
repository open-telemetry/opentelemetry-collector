// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// MergeSplit splits and/or merges the provided traces request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *tracesRequest) MergeSplit(_ context.Context, maxSize int, szt request.SizerType, r2 request.Request) ([]request.Request, error) {
	var sz sizer.TracesSizer
	switch szt {
	case request.SizerTypeItems:
		sz = &sizer.TracesCountSizer{}
	case request.SizerTypeBytes:
		sz = &sizer.TracesBytesSizer{}
	default:
		return nil, errors.New("unknown sizer type")
	}

	if r2 != nil {
		req2, ok := r2.(*tracesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sz, szt)
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []request.Request{req}, nil
	}
	return req.split(maxSize, sz, szt)
}

func (req *tracesRequest) mergeTo(dst *tracesRequest, sz sizer.TracesSizer, szt request.SizerType) {
	if sz != nil {
		dst.sizes.Update(szt, dst.size(sz, szt)+req.size(sz, szt))
		req.sizes.Update(szt, 0)
	}
	req.td.ResourceSpans().MoveAndAppendTo(dst.td.ResourceSpans())
}

func (req *tracesRequest) split(maxSize int, sz sizer.TracesSizer, szt request.SizerType) ([]request.Request, error) {
	var res []request.Request
	droppedItems := 0
	unsplittable := false
	for req.size(sz, szt) > maxSize {
		td, rmSize := extractTraces(req.td, maxSize, sz)
		if td.SpanCount() == 0 {
			// The next span does not fit into maxSize even on its own, so no batch
			// can ever hold it. Drop only that span and keep splitting the rest,
			// otherwise every remaining span is discarded along with it.
			if !removeFirstSpan(req.td) {
				// There is no span left to drop, yet the request is still over maxSize,
				// so its resource and scope overhead alone exceeds the limit. Stop
				// instead of looping forever, and report it below rather than reporting
				// success for a request that was never split.
				unsplittable = true
				break
			}
			droppedItems++
			req.sizes.Update(szt, sz.TracesSize(req.td))
			continue
		}
		req.sizes.Update(szt, req.size(sz, szt)-rmSize)
		res = append(res, newTracesRequest(td))
	}
	if unsplittable {
		// removeFirstSpan prunes the scopes and resources it empties even when it finds
		// no span to remove, so the cached size is stale by this point.
		req.sizes.Update(szt, sz.TracesSize(req.td))
	}
	// Keep the remainder, unless splitting emptied it, in which case there is
	// nothing left to export.
	if (droppedItems == 0 && !unsplittable) || req.td.SpanCount() > 0 {
		res = append(res, req)
	}
	switch {
	case droppedItems > 0:
		return res, fmt.Errorf("one span size is greater than max size, dropping items: %d", droppedItems)
	case unsplittable:
		return res, errors.New("request size is greater than max size and has no spans left to drop")
	}
	return res, nil
}

// removeFirstSpan removes the first span in iteration order, together with the
// scope and resource that it leaves empty. Reports whether a span was removed,
// which is false only when there are none left.
func removeFirstSpan(td ptrace.Traces) bool {
	removed := false
	td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool {
		if removed {
			return false
		}
		rs.ScopeSpans().RemoveIf(func(ss ptrace.ScopeSpans) bool {
			if removed {
				return false
			}
			ss.Spans().RemoveIf(func(ptrace.Span) bool {
				if removed {
					return false
				}
				removed = true
				return true
			})
			return ss.Spans().Len() == 0
		})
		return rs.ScopeSpans().Len() == 0
	})
	return removed
}

// extractTraces extracts a new traces with a maximum number of spans.
func extractTraces(srcTraces ptrace.Traces, capacity int, sz sizer.TracesSizer) (ptrace.Traces, int) {
	destTraces := ptrace.NewTraces()
	capacityLeft := capacity - sz.TracesSize(destTraces)
	removedSize := 0
	srcTraces.ResourceSpans().RemoveIf(func(srcRS ptrace.ResourceSpans) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRsSize := sz.ResourceSpansSize(srcRS)
		rsSize := sz.DeltaSize(rawRsSize)

		if rsSize > capacityLeft {
			extSrcRS, extRsSize := extractResourceSpans(srcRS, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRsSize
			// There represents the delta between the delta sizes.
			removedSize += rsSize - rawRsSize - (sz.DeltaSize(rawRsSize-extRsSize) - (rawRsSize - extRsSize))
			// It is possible that for the bytes scenario, the extracted field contains no spans.
			// Do not add it to the destination if that is the case.
			if extSrcRS.ScopeSpans().Len() > 0 {
				extSrcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
			}
			return extSrcRS.ScopeSpans().Len() != 0
		}
		capacityLeft -= rsSize
		removedSize += rsSize

		srcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
		return true
	})
	return destTraces, removedSize
}

// extractResourceSpans extracts spans and returns a new resource spans with the specified number of spans.
func extractResourceSpans(srcRS ptrace.ResourceSpans, capacity int, sz sizer.TracesSizer) (ptrace.ResourceSpans, int) {
	destRS := ptrace.NewResourceSpans()
	destRS.SetSchemaUrl(srcRS.SchemaUrl())
	srcRS.Resource().CopyTo(destRS.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceSpansSize(destRS)
	removedSize := 0
	srcRS.ScopeSpans().RemoveIf(func(srcSS ptrace.ScopeSpans) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rawSlSize := sz.ScopeSpansSize(srcSS)
		ssSize := sz.DeltaSize(rawSlSize)
		if ssSize > capacityLeft {
			extSrcSS, extSsSize := extractScopeSpans(srcSS, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extSsSize
			// There represents the delta between the delta sizes.
			removedSize += ssSize - rawSlSize - (sz.DeltaSize(rawSlSize-extSsSize) - (rawSlSize - extSsSize))
			// It is possible that for the bytes scenario, the extracted field contains no spans.
			// Do not add it to the destination if that is the case.
			if extSrcSS.Spans().Len() > 0 {
				extSrcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
			}
			return extSrcSS.Spans().Len() != 0
		}
		capacityLeft -= ssSize
		removedSize += ssSize

		srcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
		return true
	})
	return destRS, removedSize
}

// extractScopeSpans extracts spans and returns a new scope spans with the specified number of spans.
func extractScopeSpans(srcSS ptrace.ScopeSpans, capacity int, sz sizer.TracesSizer) (ptrace.ScopeSpans, int) {
	destSS := ptrace.NewScopeSpans()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeSpansSize(destSS)
	removedSize := 0
	srcSS.Spans().RemoveIf(func(srcSpan ptrace.Span) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rsSize := sz.DeltaSize(sz.SpanSize(srcSpan))
		if rsSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}

		capacityLeft -= rsSize
		removedSize += rsSize
		srcSpan.MoveTo(destSS.Spans().AppendEmpty())
		return true
	})
	return destSS, removedSize
}
