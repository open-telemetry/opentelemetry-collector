// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Merge merges the provided traces request into the current request and returns the merged request.
func (req *tracesRequest) Merge(_ context.Context, r2 Request) (Request, error) {
	tr2, ok2 := r2.(*tracesRequest)
	if !ok2 {
		return nil, errors.New("invalid input type")
	}
	tr2.td.ResourceSpans().MoveAndAppendTo(req.td.ResourceSpans())
	return req, nil
}

// MergeSplit splits and/or merges the provided traces request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *tracesRequest) MergeSplit(_ context.Context, cfg exporterbatcher.MaxSizeConfig, r2 Request) ([]Request, error) {
	var (
		res          []Request
		destReq      *tracesRequest
		capacityLeft = cfg.MaxSizeItems
	)
	for _, req := range []Request{req, r2} {
		if req == nil {
			continue
		}
		srcReq, ok := req.(*tracesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		if srcReq.td.SpanCount() <= capacityLeft {
			if destReq == nil {
				destReq = srcReq
			} else {
				srcReq.td.ResourceSpans().MoveAndAppendTo(destReq.td.ResourceSpans())
			}
			capacityLeft -= destReq.td.SpanCount()
			continue
		}

		for {
			extractedTraces := extractTraces(srcReq.td, capacityLeft)
			if extractedTraces.SpanCount() == 0 {
				break
			}
			capacityLeft -= extractedTraces.SpanCount()
			if destReq == nil {
				destReq = &tracesRequest{td: extractedTraces, pusher: srcReq.pusher}
			} else {
				extractedTraces.ResourceSpans().MoveAndAppendTo(destReq.td.ResourceSpans())
			}
			// Create new batch once capacity is reached.
			if capacityLeft == 0 {
				res = append(res, destReq)
				destReq = nil
				capacityLeft = cfg.MaxSizeItems
			}
		}
	}

	if destReq != nil {
		res = append(res, destReq)
	}
	return res, nil
}

// extractTraces extracts a new traces with a maximum number of spans.
func extractTraces(srcTraces ptrace.Traces, count int) ptrace.Traces {
	destTraces := ptrace.NewTraces()
	srcTraces.ResourceSpans().RemoveIf(func(srcRS ptrace.ResourceSpans) bool {
		if count == 0 {
			return false
		}
		needToExtract := resourceTracesCount(srcRS) > count
		if needToExtract {
			srcRS = extractResourceSpans(srcRS, count)
		}
		count -= resourceTracesCount(srcRS)
		srcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
		return !needToExtract
	})
	return destTraces
}

// extractResourceSpans extracts spans and returns a new resource spans with the specified number of spans.
func extractResourceSpans(srcRS ptrace.ResourceSpans, count int) ptrace.ResourceSpans {
	destRS := ptrace.NewResourceSpans()
	destRS.SetSchemaUrl(srcRS.SchemaUrl())
	srcRS.Resource().CopyTo(destRS.Resource())
	srcRS.ScopeSpans().RemoveIf(func(srcSS ptrace.ScopeSpans) bool {
		if count == 0 {
			return false
		}
		needToExtract := srcSS.Spans().Len() > count
		if needToExtract {
			srcSS = extractScopeSpans(srcSS, count)
		}
		count -= srcSS.Spans().Len()
		srcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
		return !needToExtract
	})
	srcRS.Resource().CopyTo(destRS.Resource())
	return destRS
}

// extractScopeSpans extracts spans and returns a new scope spans with the specified number of spans.
func extractScopeSpans(srcSS ptrace.ScopeSpans, count int) ptrace.ScopeSpans {
	destSS := ptrace.NewScopeSpans()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	srcSS.Spans().RemoveIf(func(srcSpan ptrace.Span) bool {
		if count == 0 {
			return false
		}
		srcSpan.MoveTo(destSS.Spans().AppendEmpty())
		count--
		return true
	})
	return destSS
}

// resourceTracesCount calculates the total number of spans in the pdata.ResourceSpans.
func resourceTracesCount(rs ptrace.ResourceSpans) int {
	count := 0
	rs.ScopeSpans().RemoveIf(func(ss ptrace.ScopeSpans) bool {
		count += ss.Spans().Len()
		return false
	})
	return count
}
