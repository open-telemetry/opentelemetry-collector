// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batchprocessor

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// splitTrace removes spans from the input trace and returns a new trace of the specified size.
func splitTrace(size int, toSplit pdata.Traces) pdata.Traces {
	if toSplit.SpanCount() <= size {
		return toSplit
	}
	copiedSpans := 0
	result := pdata.NewTraces()
	rss := toSplit.ResourceSpans()
	for i := rss.Len() - 1; i >= 0; i-- {
		rs := rss.At(i)
		destRs := pdata.NewResourceSpans()
		rs.Resource().CopyTo(destRs.Resource())
		result.ResourceSpans().Append(destRs)

		for j := rs.InstrumentationLibrarySpans().Len() - 1; j >= 0; j-- {
			instSpans := rs.InstrumentationLibrarySpans().At(j)
			destInstSpans := pdata.NewInstrumentationLibrarySpans()
			destRs.InstrumentationLibrarySpans().Append(destInstSpans)
			instSpans.InstrumentationLibrary().CopyTo(destInstSpans.InstrumentationLibrary())

			if size-copiedSpans >= instSpans.Spans().Len() {
				destInstSpans.Spans().Resize(instSpans.Spans().Len())
			} else {
				destInstSpans.Spans().Resize(size - copiedSpans)
			}
			for k, destIdx := instSpans.Spans().Len()-1, 0; k >= 0 && copiedSpans < size; k, destIdx = k-1, destIdx+1 {
				span := instSpans.Spans().At(k)
				span.CopyTo(destInstSpans.Spans().At(destIdx))
				copiedSpans++
				// remove span
				instSpans.Spans().Resize(instSpans.Spans().Len() - 1)
			}
			if instSpans.Spans().Len() == 0 {
				rs.InstrumentationLibrarySpans().Resize(rs.InstrumentationLibrarySpans().Len() - 1)
			}
			if copiedSpans == size {
				return result
			}
		}
		if rs.InstrumentationLibrarySpans().Len() == 0 {
			rss.Resize(rss.Len() - 1)
		}
	}
	return result
}
