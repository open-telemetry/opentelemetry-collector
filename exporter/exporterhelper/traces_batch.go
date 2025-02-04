// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// MergeSplit splits and/or merges the provided traces request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *tracesRequest) MergeSplit(_ context.Context, cfg exporterbatcher.MaxSizeConfig, r2 Request) ([]Request, error) {
	if r2 != nil {
		req2, ok := r2.(*tracesRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req)
	}

	// If no limit we can simply merge the new request into the current and return.
	if cfg.MaxSizeItems == 0 {
		return []Request{req}, nil
	}
	return req.split(cfg)
}

func (req *tracesRequest) mergeTo(dst *tracesRequest) {
	dst.setCachedItemsCount(dst.ItemsCount() + req.ItemsCount())
	req.setCachedItemsCount(0)
	req.td.ResourceSpans().MoveAndAppendTo(dst.td.ResourceSpans())
}

func (req *tracesRequest) split(cfg exporterbatcher.MaxSizeConfig) ([]Request, error) {
	var res []Request
	for req.ItemsCount() > cfg.MaxSizeItems {
		td := extractTraces(req.td, cfg.MaxSizeItems)
		size := td.SpanCount()
		req.setCachedItemsCount(req.ItemsCount() - size)
		res = append(res, &tracesRequest{td: td, pusher: req.pusher, cachedItemsCount: size})
	}
	res = append(res, req)
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
