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
func splitMetrics(size int, toSplit pdata.Metrics) pdata.Metrics {
	if toSplit.MetricCount() <= size {
		return toSplit
	}
	copiedMetrics := 0
	result := pdata.NewMetrics()
	result.ResourceMetrics().Resize(toSplit.ResourceMetrics().Len())
	rms := toSplit.ResourceMetrics()

	rmsCount := 0
	for i := rms.Len() - 1; i >= 0; i-- {
		rmsCount++
		rm := rms.At(i)
		destRs := result.ResourceMetrics().At(result.ResourceMetrics().Len() - 1 - i)
		rm.Resource().CopyTo(destRs.Resource())

		destRs.InstrumentationLibraryMetrics().Resize(rm.InstrumentationLibraryMetrics().Len())

		ilmCount := 0
		for j := rm.InstrumentationLibraryMetrics().Len() - 1; j >= 0; j-- {
			ilmCount++
			instMetrics := rm.InstrumentationLibraryMetrics().At(j)
			destInstMetrics := destRs.InstrumentationLibraryMetrics().At(destRs.InstrumentationLibraryMetrics().Len() - 1 - j)
			instMetrics.InstrumentationLibrary().CopyTo(destInstMetrics.InstrumentationLibrary())

			if size-copiedMetrics >= instMetrics.Metrics().Len() {
				destInstMetrics.Metrics().Resize(instMetrics.Metrics().Len())
			} else {
				destInstMetrics.Metrics().Resize(size - copiedMetrics)
			}
			for k, destIdx := instMetrics.Metrics().Len()-1, 0; k >= 0 && copiedMetrics < size; k, destIdx = k-1, destIdx+1 {
				metric := instMetrics.Metrics().At(k)
				metric.CopyTo(destInstMetrics.Metrics().At(destIdx))
				copiedMetrics++
				// remove metric
				instMetrics.Metrics().Resize(instMetrics.Metrics().Len() - 1)
			}
			if instMetrics.Metrics().Len() == 0 {
				rm.InstrumentationLibraryMetrics().Resize(rm.InstrumentationLibraryMetrics().Len() - 1)
			}
			if copiedMetrics == size {
				result.ResourceMetrics().Resize(rmsCount)
				return result
			}
		}
		destRs.InstrumentationLibraryMetrics().Resize(ilmCount)
		if rm.InstrumentationLibraryMetrics().Len() == 0 {
			rms.Resize(rms.Len() - 1)
		}
	}
	result.ResourceMetrics().Resize(rmsCount)
	return result
}
