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

type metric struct {
	md   data.MetricData
	seen bool
}

type metricIndex struct {
	metricsByName map[string]*metric
}

func NewMetricIndex(mds []data.MetricData) *metricIndex {
	mi := &metricIndex{metricsByName: map[string]*metric{}}
	for _, md := range mds {
		metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
		name := metrics.At(0).MetricDescriptor().Name()
		mi.metricsByName[name] = &metric{md: md}
	}
	return mi
}

func (mi *metricIndex) lookup(name string) (*metric, bool) {
	md, ok := mi.metricsByName[name]
	return md, ok
}

func (mi *metricIndex) allSeen() bool {
	for _, m := range mi.metricsByName {
		if !m.seen {
			return false
		}
	}
	return true
}
