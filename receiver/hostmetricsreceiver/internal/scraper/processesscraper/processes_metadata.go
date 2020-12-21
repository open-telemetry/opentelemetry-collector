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

// labels

const (
	statusLabelName = "status"
)

// status label values

const (
	blockedStatusLabelValue = "blocked"
	runningStatusLabelValue = "running"
)

// descriptors

var processesCreatedDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.processes.created")
	metric.SetDescription("Total number of created processes.")
	metric.SetUnit("{processes}")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var processesCountDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.processes.count")
	metric.SetDescription("Total number of processes in each state.")
	metric.SetUnit("{processes}")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
