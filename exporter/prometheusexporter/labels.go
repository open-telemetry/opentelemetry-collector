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

package prometheusexporter

import (
	"sort"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type labelKeys struct {
	// ordered OC label keys
	keys []string
	// map from a label key literal
	// to its index in the slice above
	keyIndices map[string]int
}

func collectLabelKeys(metric pdata.Metric) *labelKeys {
	// First, collect a set of all labels present in the metric
	keySet := make(map[string]struct{})

	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		collectLabelKeysIntDataPoints(metric.IntGauge().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleGauge:
		collectLabelKeysDoubleDataPoints(metric.DoubleGauge().DataPoints(), keySet)
	case pdata.MetricDataTypeIntSum:
		collectLabelKeysIntDataPoints(metric.IntSum().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleSum:
		collectLabelKeysDoubleDataPoints(metric.DoubleSum().DataPoints(), keySet)
	case pdata.MetricDataTypeIntHistogram:
		collectLabelKeysIntHistogramDataPoints(metric.IntHistogram().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleHistogram:
		collectLabelKeysDoubleHistogramDataPoints(metric.DoubleHistogram().DataPoints(), keySet)
	}

	if len(keySet) == 0 {
		return &labelKeys{}
	}

	// Sort keys
	sortedKeys := make([]string, 0, len(keySet))
	for key := range keySet {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	keys := make([]string, 0, len(sortedKeys))
	// Label values will have to match keys by index
	// so this map will help with fast lookups.
	indices := make(map[string]int, len(sortedKeys))
	for i, key := range sortedKeys {
		keys = append(keys, key)
		indices[key] = i
	}

	return &labelKeys{
		keys:       keys,
		keyIndices: indices,
	}
}

func collectLabelKeysIntDataPoints(ips pdata.IntDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < ips.Len(); i++ {
		ip := ips.At(i)
		addLabelKeys(keySet, ip.LabelsMap())
	}
}

func collectLabelKeysDoubleDataPoints(dps pdata.DoubleDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		addLabelKeys(keySet, dp.LabelsMap())
	}
}

func collectLabelKeysIntHistogramDataPoints(ihdp pdata.IntHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < ihdp.Len(); i++ {
		hp := ihdp.At(i)
		addLabelKeys(keySet, hp.LabelsMap())
	}
}

func collectLabelKeysDoubleHistogramDataPoints(dhdp pdata.DoubleHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		hp := dhdp.At(i)
		addLabelKeys(keySet, hp.LabelsMap())
	}
}

func addLabelKeys(keySet map[string]struct{}, labels pdata.StringMap) {
	labels.ForEach(func(k string, v string) {
		keySet[k] = struct{}{}
	})
}

func collectLabelValues(labels pdata.StringMap, lk *labelKeys) []string {
	if len(lk.keys) == 0 {
		return nil
	}

	labelValues := make([]string, len(lk.keys))
	labels.ForEach(func(k string, v string) {
		keyIndex := lk.keyIndices[k]
		labelValues[keyIndex] = v
	})

	return labelValues
}
