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

package filesystemscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// labels

const (
	deviceLabelName     = "device"
	mountModeLabelName  = "mode"
	mountPointLabelName = "mountpoint"
	stateLabelName      = "state"
	typeLabelName       = "type"
)

// state label values

const (
	freeLabelValue     = "free"
	reservedLabelValue = "reserved"
	usedLabelValue     = "used"
)

// descriptors

var fileSystemUsageDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.filesystem.usage")
	metric.SetDescription("Filesystem bytes used.")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var fileSystemINodesUsageDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.filesystem.inodes.usage")
	metric.SetDescription("FileSystem iNodes used.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
