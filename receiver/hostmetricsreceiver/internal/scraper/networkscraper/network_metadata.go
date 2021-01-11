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

package networkscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// labels

const (
	deviceLabelName    = "device"
	directionLabelName = "direction"
	protocolLabelName  = "protocol"
	stateLabelName     = "state"
)

// direction label values

const (
	receiveDirectionLabelValue  = "receive"
	transmitDirectionLabelValue = "transmit"
)

// protocol label values

const (
	tcpProtocolLabelValue = "tcp"
)

// descriptors

var networkPacketsDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.network.packets")
	metric.SetDescription("The number of packets transferred.")
	metric.SetUnit("{packets}")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var networkDroppedPacketsDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.network.dropped")
	metric.SetDescription("The number of packets dropped.")
	metric.SetUnit("{packets}")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var networkErrorsDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.network.errors")
	metric.SetDescription("The number of errors encountered")
	metric.SetUnit("{errors}")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var networkIODescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.network.io")
	metric.SetDescription("The number of bytes transmitted and received")
	metric.SetUnit("By")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var networkConnectionsDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.network.connections")
	metric.SetDescription("The number of connections")
	metric.SetUnit("{connections}")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
