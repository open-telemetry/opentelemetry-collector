// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterprocessor

import (
	"sort"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// metricIndex holds paths to metrics in a pdata.Metrics struct via the indexes
// ResourceMetrics -> InstrumentationLibraryMetrics -> Metrics. Once these
// indexes are populated, you can extract a pdata.Metrics from an existing
// pdata.Metrics with just the metrics at the specified paths. The motivation
// for this type is to allow the output of filtered metrics to not contain
// parent structs (InstrumentationLibrary, Resource, etc.) for a MetricSlice
// that has become empty after filtering.
type metricIndex struct {
	m map[int]map[int]map[int]bool
}

func newMetricIndex() *metricIndex {
	return &metricIndex{m: map[int]map[int]map[int]bool{}}
}

func (idx metricIndex) add(rmIdx, ilmIdx, mIdx int) {
	rmMap, ok := idx.m[rmIdx]
	if !ok {
		rmMap = map[int]map[int]bool{}
		idx.m[rmIdx] = rmMap
	}
	ilmMap, ok := rmMap[ilmIdx]
	if !ok {
		ilmMap = map[int]bool{}
		rmMap[ilmIdx] = ilmMap
	}
	ilmMap[mIdx] = true
}

func (idx metricIndex) extract(pdm pdata.Metrics) pdata.Metrics {
	out := pdata.NewMetrics()
	rmSliceOut := out.ResourceMetrics()

	sortRMIdx := sortRM(idx.m)
	rmsIn := pdm.ResourceMetrics()
	rmSliceOut.Resize(len(sortRMIdx))
	pos := 0
	for _, rmIdx := range sortRMIdx {
		rmIn := rmsIn.At(rmIdx)
		ilmSliceIn := rmIn.InstrumentationLibraryMetrics()

		rmOut := rmSliceOut.At(pos)
		pos++
		rmIn.Resource().CopyTo(rmOut.Resource())
		ilmSliceOut := rmOut.InstrumentationLibraryMetrics()
		ilmIndexes := idx.m[rmIdx]
		for _, ilmIdx := range sortILM(ilmIndexes) {
			ilmIn := ilmSliceIn.At(ilmIdx)
			mSliceIn := ilmIn.Metrics()

			ilmOut := pdata.NewInstrumentationLibraryMetrics()
			ilmSliceOut.Append(ilmOut)
			ilOut := ilmOut.InstrumentationLibrary()
			ilmIn.InstrumentationLibrary().CopyTo(ilOut)
			mSliceOut := ilmOut.Metrics()
			for _, metricIdx := range sortMetrics(ilmIndexes[ilmIdx]) {
				mSliceOut.Append(mSliceIn.At(metricIdx))
			}
		}
	}
	return out
}

func sortRM(rmIndexes map[int]map[int]map[int]bool) []int {
	var rmSorted = make([]int, len(rmIndexes))
	i := 0
	for key := range rmIndexes {
		rmSorted[i] = key
		i++
	}
	sort.Ints(rmSorted)
	return rmSorted
}

func sortILM(ilmIndexes map[int]map[int]bool) []int {
	var ilmSorted = make([]int, len(ilmIndexes))
	i := 0
	for key := range ilmIndexes {
		ilmSorted[i] = key
		i++
	}
	sort.Ints(ilmSorted)
	return ilmSorted
}

func sortMetrics(metricIndexes map[int]bool) []int {
	var metricIdxSorted = make([]int, len(metricIndexes))
	i := 0
	for key := range metricIndexes {
		metricIdxSorted[i] = key
		i++
	}
	sort.Ints(metricIdxSorted)
	return metricIdxSorted
}

func (idx metricIndex) isEmpty() bool {
	return len(idx.m) == 0
}
