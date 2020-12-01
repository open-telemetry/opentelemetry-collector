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

package processesscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// descriptors

var processesRunningDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.processes.running")
	metric.SetDescription("Total number of running processes.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var processesBlockedDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.processes.blocked")
	metric.SetDescription("Total number of blocked processes.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
