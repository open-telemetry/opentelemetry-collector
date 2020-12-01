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

package processscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// labels

const (
	directionLabelName = "direction"
	stateLabelName     = "state"
)

// direction label values

const (
	readDirectionLabelValue  = "read"
	writeDirectionLabelValue = "write"
)

// state label values

const (
	userStateLabelValue   = "user"
	systemStateLabelValue = "system"
	waitStateLabelValue   = "wait"
)

// descriptors

var cpuTimeDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("process.cpu.time")
	metric.SetDescription("Total CPU seconds broken down by different states.")
	metric.SetUnit("s")
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	sum := metric.DoubleSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var physicalMemoryUsageDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("process.memory.physical_usage")
	metric.SetDescription("The amount of physical memory in use.")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var virtualMemoryUsageDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("process.memory.virtual_usage")
	metric.SetDescription("Virtual memory size.")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var diskIODescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("process.disk.io")
	metric.SetDescription("Disk bytes transferred.")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
