// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// splitTraces removes spans from the input trace and returns a new trace of the specified size.
func splitTraces(size int, src ptrace.Traces) ptrace.Traces {
	if !hasAtLeastNSpans(src, size+1) {
		return src
	}
	totalCopiedSpans := 0
	dest := ptrace.NewTraces()

	src.ResourceSpans().RemoveIf(func(srcRs ptrace.ResourceSpans) bool {
		// If we are done skip everything else.
		if totalCopiedSpans == size {
			return false
		}

		// If it fully fits
		wontFit, srcRsSC := resourceHasAtLeastNSpans(srcRs, size-totalCopiedSpans+1)
		if !wontFit {
			totalCopiedSpans += srcRsSC
			srcRs.MoveTo(dest.ResourceSpans().AppendEmpty())
			return true
		}

		destRs := dest.ResourceSpans().AppendEmpty()
		srcRs.Resource().CopyTo(destRs.Resource())
		srcRs.ScopeSpans().RemoveIf(func(srcIls ptrace.ScopeSpans) bool {
			// If we are done skip everything else.
			if totalCopiedSpans == size {
				return false
			}

			// If possible to move all metrics do that.
			srcIlsSC := srcIls.Spans().Len()
			if size-totalCopiedSpans >= srcIlsSC {
				totalCopiedSpans += srcIlsSC
				srcIls.MoveTo(destRs.ScopeSpans().AppendEmpty())
				return true
			}

			destIls := destRs.ScopeSpans().AppendEmpty()
			srcIls.Scope().CopyTo(destIls.Scope())
			srcIls.Spans().RemoveIf(func(srcSpan ptrace.Span) bool {
				// If we are done skip everything else.
				if totalCopiedSpans == size {
					return false
				}
				srcSpan.MoveTo(destIls.Spans().AppendEmpty())
				totalCopiedSpans++
				return true
			})
			return false
		})
		return srcRs.ScopeSpans().Len() == 0
	})

	return dest
}

func resourceHasAtLeastNSpans(rs ptrace.ResourceSpans, n int) (bool, int) {
	count := 0
	for k := 0; k < rs.ScopeSpans().Len(); k++ {
		count += rs.ScopeSpans().At(k).Spans().Len()
		if count >= n {
			return true, 0
		}
	}
	return false, count
}

func hasAtLeastNSpans(ms ptrace.Traces, n int) bool {
	spanCount := 0
	rss := ms.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			spanCount += ilss.At(j).Spans().Len()
			if spanCount >= n {
				return true
			}
		}
	}
	return spanCount >= n
}
