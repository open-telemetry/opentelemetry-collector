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

// splitMetrics removes metrics from the input data and returns a new data of the specified size.
func splitMetrics(size int, src pdata.Metrics) pdata.Metrics {
	if src.MetricCount() <= size {
		return src
	}
	totalCopiedMetrics := 0
	dest := pdata.NewMetrics()

	src.ResourceMetrics().RemoveIf(func(srcRm pdata.ResourceMetrics) bool {
		// If we are done skip everything else.
		if totalCopiedMetrics == size {
			return false
		}

		destRs := dest.ResourceMetrics().AppendEmpty()
		srcRm.Resource().CopyTo(destRs.Resource())

		srcRm.InstrumentationLibraryMetrics().RemoveIf(func(srcIm pdata.InstrumentationLibraryMetrics) bool {
			// If we are done skip everything else.
			if totalCopiedMetrics == size {
				return false
			}

			destIms := destRs.InstrumentationLibraryMetrics().AppendEmpty()
			srcIm.InstrumentationLibrary().CopyTo(destIms.InstrumentationLibrary())

			// If possible to move all metrics do that.
			srcMetricsLen := srcIm.Metrics().Len()
			if size-totalCopiedMetrics >= srcMetricsLen {
				totalCopiedMetrics += srcMetricsLen
				srcIm.Metrics().MoveAndAppendTo(destIms.Metrics())
				return true
			}

			srcIm.Metrics().RemoveIf(func(srcMetric pdata.Metric) bool {
				// If we are done skip everything else.
				if totalCopiedMetrics == size {
					return false
				}
				srcMetric.CopyTo(destIms.Metrics().AppendEmpty())
				totalCopiedMetrics++
				return true
			})
			return false
		})
		return srcRm.InstrumentationLibraryMetrics().Len() == 0
	})

	return dest
}
