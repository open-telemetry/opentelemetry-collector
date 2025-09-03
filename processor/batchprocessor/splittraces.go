// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// splitTraces removes spans from the input trace and returns a new trace of the specified size.
func splitTraces(size int, src ptrace.Traces) ptrace.Traces {
	if src.SpanCount() <= size {
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
		srcRsSC := resourceSC(srcRs)
		if (totalCopiedSpans + srcRsSC) <= size {
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

// resourceSC calculates the total number of spans in the ptrace.ResourceSpans.
func resourceSC(rs ptrace.ResourceSpans) (count int) {
	for k := 0; k < rs.ScopeSpans().Len(); k++ {
		count += rs.ScopeSpans().At(k).Spans().Len()
	}
	return count
}
