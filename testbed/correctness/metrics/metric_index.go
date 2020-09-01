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

package metrics

import "go.opentelemetry.io/collector/internal/data"

type metricReceived struct {
	md       data.MetricData
	received bool
}

type metricsReceivedIndex struct {
	m map[string]*metricReceived
}

func newMetricsReceivedIndex(mds []data.MetricData) *metricsReceivedIndex {
	mi := &metricsReceivedIndex{m: map[string]*metricReceived{}}
	for _, md := range mds {
		metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
		name := metrics.At(0).Name()
		mi.m[name] = &metricReceived{md: md}
	}
	return mi
}

func (mi *metricsReceivedIndex) lookup(name string) (*metricReceived, bool) {
	md, ok := mi.m[name]
	return md, ok
}

func (mi *metricsReceivedIndex) allReceived() bool {
	for _, m := range mi.m {
		if !m.received {
			return false
		}
	}
	return true
}
