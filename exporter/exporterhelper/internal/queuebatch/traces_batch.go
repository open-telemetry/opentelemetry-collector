// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// MergeSplit splits and/or merges the provided traces request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *tracesRequest) MergeSplit(_ context.Context, limits map[request.SizerType]int64, r2 request.Request) ([]request.Request, error) {
	sizers := make(map[request.SizerType]sizer.TracesSizer)
	for szt := range limits {
		switch szt {
		case request.SizerTypeItems:
			sizers[szt] = &sizer.TracesCountSizer{}
		case request.SizerTypeBytes:
			sizers[szt] = &sizer.TracesBytesSizer{}
		default:
			return nil, errors.New("unknown sizer type")
		}
	}

	if r2 != nil {
		req2, ok := r2.(*tracesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sizers)
	}

	// If no limits we can simply merge the new request into the current and return.
	if len(limits) == 0 {
		return []request.Request{req}, nil
	}
	return req.split(limits, sizers)
}

func (req *tracesRequest) mergeTo(dst *tracesRequest, sizers map[request.SizerType]sizer.TracesSizer) {
	for szt, sz := range sizers {
		dst.setCachedSize(szt, dst.size(szt, sz)+req.size(szt, sz))
		req.setCachedSize(szt, 0)
	}
	req.td.ResourceSpans().MoveAndAppendTo(dst.td.ResourceSpans())
}

func (req *tracesRequest) split(limits map[request.SizerType]int64, sizers map[request.SizerType]sizer.TracesSizer) ([]request.Request, error) {
	var res []request.Request
	for {
		exceeded := false
		for szt, limit := range limits {
			sz := sizers[szt]
			if limit > 0 && int64(req.size(szt, sz)) > limit {
				exceeded = true
				break
			}
		}
		if !exceeded {
			break
		}

		adjustedLimits := make(map[request.SizerType]int64)
		for szt, limit := range limits {
			if limit == 0 {
				adjustedLimits[szt] = math.MaxInt64
			} else {
				adjustedLimits[szt] = limit
			}
		}

		td, rmSizes := extractTraces(req.td, adjustedLimits, sizers)
		if td.SpanCount() == 0 {
			return res, fmt.Errorf("one span size is greater than max size, dropping items: %d", req.td.SpanCount())
		}
		for szt, sz := range sizers {
			req.setCachedSize(szt, req.size(szt, sz)-rmSizes[szt])
		}
		res = append(res, newTracesRequest(td))
	}
	res = append(res, req)
	return res, nil
}

// extractTraces extracts a new traces with a maximum number of spans.
func extractTraces(srcTraces ptrace.Traces, limits map[request.SizerType]int64, sizers map[request.SizerType]sizer.TracesSizer) (ptrace.Traces, map[request.SizerType]int) {
	destTraces := ptrace.NewTraces()
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = int(limit) - sz.TracesSize(destTraces)
		removedSizes[szt] = 0
	}

	srcTraces.ResourceSpans().RemoveIf(func(srcRS ptrace.ResourceSpans) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rawRsSize := sz.ResourceSpansSize(srcRS)
			rsSize := sz.DeltaSize(rawRsSize)
			if rsSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			extSrcRS, extRsSizes := extractResourceSpans(srcRS, capacityLeft, sizers)
			
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			
			for szt, sz := range sizers {
				rawRsSize := sz.ResourceSpansSize(srcRS)
				rsSize := sz.DeltaSize(rawRsSize)
				extRsSize := extRsSizes[szt]
				removedSizes[szt] += extRsSize
				removedSizes[szt] += rsSize - rawRsSize - (sz.DeltaSize(rawRsSize-extRsSize) - (rawRsSize - extRsSize))
			}

			if extSrcRS.ScopeSpans().Len() > 0 {
				extSrcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
			}
			return extSrcRS.ScopeSpans().Len() != 0
		}

		for szt, sz := range sizers {
			rawRsSize := sz.ResourceSpansSize(srcRS)
			rsSize := sz.DeltaSize(rawRsSize)
			capacityLeft[szt] -= rsSize
			removedSizes[szt] += rsSize
		}
		srcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
		return true
	})
	return destTraces, removedSizes
}

// extractResourceSpans extracts spans and returns a new resource spans with the specified number of spans.
func extractResourceSpans(srcRS ptrace.ResourceSpans, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.TracesSizer) (ptrace.ResourceSpans, map[request.SizerType]int) {
	destRS := ptrace.NewResourceSpans()
	destRS.SetSchemaUrl(srcRS.SchemaUrl())
	srcRS.Resource().CopyTo(destRS.Resource())
	
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)
	
	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.ResourceSpansSize(destRS)
		removedSizes[szt] = 0
	}

	srcRS.ScopeSpans().RemoveIf(func(srcSS ptrace.ScopeSpans) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rawSlSize := sz.ScopeSpansSize(srcSS)
			ssSize := sz.DeltaSize(rawSlSize)
			if ssSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			extSrcSS, extSsSizes := extractScopeSpans(srcSS, capacityLeft, sizers)
			
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			
			for szt, sz := range sizers {
				rawSlSize := sz.ScopeSpansSize(srcSS)
				ssSize := sz.DeltaSize(rawSlSize)
				extSsSize := extSsSizes[szt]
				removedSizes[szt] += extSsSize
				removedSizes[szt] += ssSize - rawSlSize - (sz.DeltaSize(rawSlSize-extSsSize) - (rawSlSize - extSsSize))
			}

			if extSrcSS.Spans().Len() > 0 {
				extSrcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
			}
			return extSrcSS.Spans().Len() != 0
		}

		for szt, sz := range sizers {
			rawSlSize := sz.ScopeSpansSize(srcSS)
			ssSize := sz.DeltaSize(rawSlSize)
			capacityLeft[szt] -= ssSize
			removedSizes[szt] += ssSize
		}
		srcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
		return true
	})
	return destRS, removedSizes
}

// extractScopeSpans extracts spans and returns a new scope spans with the specified number of spans.
func extractScopeSpans(srcSS ptrace.ScopeSpans, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.TracesSizer) (ptrace.ScopeSpans, map[request.SizerType]int) {
	destSS := ptrace.NewScopeSpans()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)
	
	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.ScopeSpansSize(destSS)
		removedSizes[szt] = 0
	}

	srcSS.Spans().RemoveIf(func(srcSpan ptrace.Span) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rsSize := sz.DeltaSize(sz.SpanSize(srcSpan))
			if rsSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			return false
		}

		for szt, sz := range sizers {
			rsSize := sz.DeltaSize(sz.SpanSize(srcSpan))
			capacityLeft[szt] -= rsSize
			removedSizes[szt] += rsSize
		}
		srcSpan.MoveTo(destSS.Spans().AppendEmpty())
		return true
	})
	return destSS, removedSizes
}
