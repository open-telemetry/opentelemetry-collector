// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func tracesSizerFromConfig(cfg exporterbatcher.SizeConfig) sizer.TracesSizer {
	switch cfg.Sizer {
	case exporterbatcher.SizerTypeItems:
		return &sizer.TracesCountSizer{}
	case exporterbatcher.SizerTypeBytes:
		return &sizer.TracesBytesSizer{}
	default:
		return &sizer.TracesCountSizer{}
	}
}

// MergeSplit splits and/or merges the provided traces request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *tracesRequest) MergeSplit(_ context.Context, cfg exporterbatcher.SizeConfig, r2 Request) ([]Request, error) {
	sizer := tracesSizerFromConfig(cfg)
	if r2 != nil {
		req2, ok := r2.(*tracesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sizer)
	}

	// If no limit we can simply merge the new request into the current and return.
	if cfg.MaxSize == 0 {
		return []Request{req}, nil
	}
	return req.split(cfg.MaxSize, sizer), nil
}

func (req *tracesRequest) mergeTo(dst *tracesRequest, sizer sizer.TracesSizer) {
	if sizer != nil {
		dst.setCachedSize(dst.Size(sizer) + req.Size(sizer))
		req.setCachedSize(0)
	}
	req.td.ResourceSpans().MoveAndAppendTo(dst.td.ResourceSpans())
}

func (req *tracesRequest) split(maxSize int, sizer sizer.TracesSizer) []Request {
	var res []Request
	for req.Size(sizer) > maxSize {
		td := extractTraces(req.td, maxSize, sizer)
		size := sizer.TracesSize(td)
		req.setCachedSize(req.Size(sizer) - size)
		res = append(res, &tracesRequest{td: td, pusher: req.pusher, cachedSize: size})
	}
	res = append(res, req)
	return res
}

// extractTraces extracts a new traces with a maximum number of spans.
func extractTraces(srcTraces ptrace.Traces, capacity int, sizer sizer.TracesSizer) ptrace.Traces {
	capacityReached := false
	destTraces := ptrace.NewTraces()
	capacityLeft := capacity - sizer.TracesSize(destTraces)
	srcTraces.ResourceSpans().RemoveIf(func(srcRS ptrace.ResourceSpans) bool {
		if capacityReached {
			return false
		}
		needToExtract := sizer.ResourceSpansSize(srcRS) > capacityLeft
		if needToExtract {
			srcRS, capacityReached = extractResourceSpans(srcRS, capacityLeft, sizer)
			if srcRS.ScopeSpans().Len() == 0 {
				return false
			}
		}
		capacityLeft -= sizer.DeltaSize(sizer.ResourceSpansSize(srcRS))
		srcRS.MoveTo(destTraces.ResourceSpans().AppendEmpty())
		return !needToExtract
	})
	return destTraces
}

// extractResourceSpans extracts spans and returns a new resource spans with the specified number of spans.
func extractResourceSpans(srcRS ptrace.ResourceSpans, capacity int, sizer sizer.TracesSizer) (ptrace.ResourceSpans, bool) {
	capacityReached := false
	destRS := ptrace.NewResourceSpans()
	destRS.SetSchemaUrl(srcRS.SchemaUrl())
	srcRS.Resource().CopyTo(destRS.Resource())
	capacityLeft := capacity - sizer.ResourceSpansSize(destRS)
	srcRS.ScopeSpans().RemoveIf(func(srcSS ptrace.ScopeSpans) bool {
		if capacityReached {
			return false
		}
		needToExtract := sizer.ScopeSpansSize(srcSS) > capacityLeft
		if needToExtract {
			srcSS, capacityReached = extractScopeSpans(srcSS, capacityLeft, sizer)
			if srcSS.Spans().Len() == 0 {
				return false
			}
		}
		capacityLeft -= sizer.DeltaSize(sizer.ScopeSpansSize(srcSS))
		srcSS.MoveTo(destRS.ScopeSpans().AppendEmpty())
		return !needToExtract
	})
	return destRS, capacityReached
}

// extractScopeSpans extracts spans and returns a new scope spans with the specified number of spans.
func extractScopeSpans(srcSS ptrace.ScopeSpans, capacity int, sizer sizer.TracesSizer) (ptrace.ScopeSpans, bool) {
	capacityReached := false
	destSS := ptrace.NewScopeSpans()
	destSS.SetSchemaUrl(srcSS.SchemaUrl())
	srcSS.Scope().CopyTo(destSS.Scope())
	capacityLeft := capacity - sizer.ScopeSpansSize(destSS)
	srcSS.Spans().RemoveIf(func(srcSpan ptrace.Span) bool {
		if capacityReached || sizer.SpanSize(srcSpan) > capacityLeft {
			capacityReached = true
			return false
		}
		capacityLeft -= sizer.DeltaSize(sizer.SpanSize(srcSpan))

		srcSpan.MoveTo(destSS.Spans().AppendEmpty())
		return true
	})
	return destSS, capacityReached
}
