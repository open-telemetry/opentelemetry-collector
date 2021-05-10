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

	src.ResourceMetrics().RemoveIf(func(srcRs pdata.ResourceMetrics) bool {
		// If we are done skip everything else.
		if totalCopiedMetrics == size {
			return false
		}

		destRs := dest.ResourceMetrics().AppendEmpty()
		srcRs.Resource().CopyTo(destRs.Resource())

		srcRs.InstrumentationLibraryMetrics().RemoveIf(func(srcIlm pdata.InstrumentationLibraryMetrics) bool {
			// If we are done skip everything else.
			if totalCopiedMetrics == size {
				return false
			}

			destIlm := destRs.InstrumentationLibraryMetrics().AppendEmpty()
			srcIlm.InstrumentationLibrary().CopyTo(destIlm.InstrumentationLibrary())

			// If possible to move all metrics do that.
			srcMetricsLen := srcIlm.Metrics().Len()
			if size-totalCopiedMetrics >= srcMetricsLen {
				totalCopiedMetrics += srcMetricsLen
				srcIlm.Metrics().MoveAndAppendTo(destIlm.Metrics())
				return true
			}

			srcIlm.Metrics().RemoveIf(func(srcMetric pdata.Metric) bool {
				// If we are done skip everything else.
				if totalCopiedMetrics == size {
					return false
				}
				srcMetric.CopyTo(destIlm.Metrics().AppendEmpty())
				totalCopiedMetrics++
				return true
			})
			return false
		})
		return srcRs.InstrumentationLibraryMetrics().Len() == 0
	})

	return dest
}
