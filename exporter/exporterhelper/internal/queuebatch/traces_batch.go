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
	if req.size(sz, szt) <= maxSize {
		return []request.Request{req}, nil
	}
	var res []request.Request
	droppedItems := 0
	for req.size(sz, szt) > maxSize {
		spansBefore := req.td.SpanCount()
		td, removedSize := extractTraces(req.td, maxSize, sz)
		if removedSize == 0 {
			// Nothing left the source, so no progress is possible. Stop rather than loop.
			return res, fmt.Errorf("request size is greater than max size and cannot be split further, dropping items: %d", droppedItems+req.td.SpanCount())
		}
		req.sizes.Update(szt, req.size(sz, szt)-removedSize)
		droppedItems += spansBefore - req.td.SpanCount() - td.SpanCount()
		if td.SpanCount() > 0 {
			res = append(res, newTracesRequest(td))
		}
	}
	// Splitting can leave nothing to export once oversized spans and span-less resources are gone.
	if req.td.SpanCount() > 0 {
		res = append(res, req)
	}
	if droppedItems > 0 {
		return res, fmt.Errorf("single span exceeds the max size limit, dropping items: %d", droppedItems)
	}
	return res, nil
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
			extSrcRS, extRsSize := extractResourceSpans(srcRS, capacityLeft, capacity, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no spans.
			// Do not add it to the destination if that is the case.
			if extSrcRS.ScopeSpans().Len() > 0 {
				extSrcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
			}
			if srcRS.ScopeSpans().Len() == 0 {
				// Nothing is left in the source resource, so all of it is removed.
				removedSize += rsSize
				return true
			}
			// The source resource shrinks to the delta size of what is left in it.
			removedSize += rsSize - sz.DeltaSize(rawRsSize-extRsSize)
			return false
		}
		capacityLeft -= rsSize
		removedSize += rsSize

		srcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
		return true
	})
	return destTraces, removedSize
}

// extractResourceSpans extracts spans and returns a new resource spans with the specified number of spans.
func extractResourceSpans(srcRS ptrace.ResourceSpans, capacity, maxSize int, sz sizer.TracesSizer) (ptrace.ResourceSpans, int) {
	destRS := ptrace.NewResourceSpans()
	destRS.SetSchemaUrl(srcRS.SchemaUrl())
	srcRS.Resource().CopyTo(destRS.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceSpansSize(destRS)
	// Room for a scope in an otherwise empty batch, once this resource's header and attributes are paid for.
	maxScopeSize := maxSize - (sz.DeltaSize(maxSize) - maxSize) - sz.ResourceSpansSize(destRS)
	removedSize := 0
	srcRS.ScopeSpans().RemoveIf(func(srcSS ptrace.ScopeSpans) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rawSsSize := sz.ScopeSpansSize(srcSS)
		ssSize := sz.DeltaSize(rawSsSize)
		if ssSize > capacityLeft {
			extSrcSS, extSsSize := extractScopeSpans(srcSS, capacityLeft, maxScopeSize, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no spans.
			// Do not add it to the destination if that is the case.
			if extSrcSS.Spans().Len() > 0 {
				extSrcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
			}
			if srcSS.Spans().Len() == 0 {
				// Nothing is left in the source scope, so all of it is removed.
				removedSize += ssSize
				return true
			}
			// The source scope shrinks to the delta size of what is left in it.
			removedSize += ssSize - sz.DeltaSize(rawSsSize-extSsSize)
			return false
		}
		capacityLeft -= ssSize
		removedSize += ssSize

		srcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
		return true
	})
	return destRS, removedSize
}

// extractScopeSpans extracts spans and returns a new scope spans with the specified number of spans.
func extractScopeSpans(srcSS ptrace.ScopeSpans, capacity, maxScopeSize int, sz sizer.TracesSizer) (ptrace.ScopeSpans, int) {
	destSS := ptrace.NewScopeSpans()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeSpansSize(destSS)
	// Largest span that fits an otherwise empty batch, once the resource and scope headers and attributes are paid for.
	maxSpanSize := maxScopeSize - (sz.DeltaSize(maxScopeSize) - maxScopeSize) - sz.ScopeSpansSize(destSS)
	removedSize := 0
	srcSS.Spans().RemoveIf(func(srcSpan ptrace.Span) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		spanSize := sz.DeltaSize(sz.SpanSize(srcSpan))
		if spanSize > maxSpanSize {
			// It can never be exported and would block every span behind it, so drop it.
			removedSize += spanSize
			return true
		}
		if spanSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}

		capacityLeft -= spanSize
		removedSize += spanSize
		srcSpan.MoveTo(destSS.Spans().AppendEmpty())
		return true
	})
	return destSS, removedSize
}
