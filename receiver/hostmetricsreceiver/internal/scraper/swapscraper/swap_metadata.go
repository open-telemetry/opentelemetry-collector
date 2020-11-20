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

package swapscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// labels

const (
	deviceLabelName    = "device"
	directionLabelName = "direction"
	stateLabelName     = "state"
	typeLabelName      = "type"
)

// direction label values

const (
	inDirectionLabelValue  = "page_in"
	outDirectionLabelValue = "page_out"
)

// state label values

const (
	cachedLabelValue = "cached"
	freeLabelValue   = "free"
	usedLabelValue   = "used"
)

// type label values

const (
	majorTypeLabelValue = "major"
	minorTypeLabelValue = "minor"
)

var swapUsageDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.swap.usage")
	metric.SetDescription("Swap (unix) or pagefile (windows) usage.")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var swapPagingDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.swap.paging_ops")
	metric.SetDescription("The number of paging operations.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()

var swapPageFaultsDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.swap.page_faults")
	metric.SetDescription("The number of page faults.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
