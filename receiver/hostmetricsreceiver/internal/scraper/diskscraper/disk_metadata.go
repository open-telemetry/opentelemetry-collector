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

package diskscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// labels

const (
	deviceLabelName    = "device"
	directionLabelName = "direction"
)

// direction label values

const (
	readDirectionLabelValue  = "read"
	writeDirectionLabelValue = "write"
)

// descriptors

var diskIODescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.disk.io")
	metric.SetDescription("Disk bytes transferred.")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var diskOpsDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.disk.ops")
	metric.SetDescription("Disk operations count.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var diskIOTimeDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.disk.io_time")
	metric.SetDescription("Time disk spent activated. On Windows, this is calculated as the inverse of disk idle time.")
	metric.SetUnit("s")
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	sum := metric.DoubleSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var diskOperationTimeDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.disk.operation_time")
	metric.SetDescription("Time spent in disk operations.")
	metric.SetUnit("s")
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	sum := metric.DoubleSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var diskPendingOperationsDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.disk.pending_operations")
	metric.SetDescription("The queue size of pending I/O operations.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var diskMergedDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.disk.merged")
	metric.SetDescription("The number of disk reads merged into single physical disk access operations.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
