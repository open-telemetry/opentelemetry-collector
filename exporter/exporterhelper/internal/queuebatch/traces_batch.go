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
	pruned := false
	unsplittable := false
	for req.size(sz, szt) > maxSize {
		td, rmSize := extractTraces(req.td, maxSize, sz)
		if td.SpanCount() == 0 {
			if td.ResourceSpans().Len() > 0 {
				// Extraction spent this batch on resources that hold no span and the
				// batch is discarded with them, but they left the source as it did so. The
				// request is smaller than when this attempt started, so try again with a
				// fresh batch instead of dropping anything: the next span may well fit.
				req.sizes.Update(szt, sz.TracesSize(req.td))
				continue
			}
			// The next span does not fit into maxSize even on its own. Drop every span in
			// that state in a single pass, rather than one per iteration with a full size
			// recompute after each, then carry on splitting what is left.
			if !pruned {
				pruned = true
				var removedAny bool
				droppedItems, removedAny = dropOversizedSpans(req.td, maxSize, sz)
				// Refresh the cache whether or not a span went: the pass also prunes the
				// scopes and resources it empties, which changes the size on its own.
				req.sizes.Update(szt, sz.TracesSize(req.td))
				if removedAny {
					continue
				}
			}
			// Nothing was extracted, nothing left the source and the pass found nothing
			// to remove, so no further progress is possible. Stop rather than loop.
			unsplittable = true
			break
		}
		req.sizes.Update(szt, req.size(sz, szt)-rmSize)
		res = append(res, newTracesRequest(td))
	}
	// Keep the remainder, unless the drop pass emptied it, in which case there is
	// nothing left to export.
	if !pruned || req.td.SpanCount() > 0 {
		res = append(res, req)
	}
	switch {
	case droppedItems > 0:
		return res, fmt.Errorf("one span size is greater than max size, dropping items: %d", droppedItems)
	case unsplittable && req.td.SpanCount() > 0:
		// Only worth reporting when a remainder is going out unsplit. With nothing left
		// in it there is no span to lose and nothing to tell the caller.
		return res, errors.New("request size is greater than max size and cannot be split further")
	}
	return res, nil
}

// dropOversizedSpans removes every span that cannot fit a batch of maxSize even on
// its own, together with the scope and resource it leaves empty, and reports how
// many it removed.
//
// A span shares each batch with its resource and scope, so their framing counts
// against maxSize too. The capacity left for a span is therefore computed the same
// way extractResourceSpans and extractScopeSpans compute it, which keeps this pass
// in step with what extraction would accept.
//
// Emptied scopes and resources are pruned as well, which shrinks the request without
// dropping a span, so the caller is told separately whether anything was removed
// rather than inferring it from the count.
func dropOversizedSpans(td ptrace.Traces, maxSize int, sz sizer.TracesSizer) (droppedItems int, removedAny bool) {
	dropped := 0
	batchCapacity := maxSize - sz.TracesSize(ptrace.NewTraces())
	td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool {
		bareRS := ptrace.NewResourceSpans()
		bareRS.SetSchemaUrl(rs.SchemaUrl())
		rs.Resource().CopyTo(bareRS.Resource())
		scopeCapacity := batchCapacity - (sz.DeltaSize(batchCapacity) - batchCapacity) - sz.ResourceSpansSize(bareRS)
		rs.ScopeSpans().RemoveIf(func(ss ptrace.ScopeSpans) bool {
			bareSS := ptrace.NewScopeSpans()
			bareSS.SetSchemaUrl(ss.SchemaUrl())
			ss.Scope().CopyTo(bareSS.Scope())
			spanCapacity := scopeCapacity - (sz.DeltaSize(scopeCapacity) - scopeCapacity) - sz.ScopeSpansSize(bareSS)
			ss.Spans().RemoveIf(func(span ptrace.Span) bool {
				if sz.DeltaSize(sz.SpanSize(span)) > spanCapacity {
					dropped++
					removedAny = true
					return true
				}
				return false
			})
			if ss.Spans().Len() == 0 {
				removedAny = true
				return true
			}
			return false
		})
		if rs.ScopeSpans().Len() == 0 {
			removedAny = true
			return true
		}
		return false
	})
	return dropped, removedAny
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
