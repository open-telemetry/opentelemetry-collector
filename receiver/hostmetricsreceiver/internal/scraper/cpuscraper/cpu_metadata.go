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

package cpuscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// labels

const (
	cpuLabelName   = "cpu"
	stateLabelName = "state"
)

// state label values

const (
	idleStateLabelValue      = "idle"
	interruptStateLabelValue = "interrupt"
	niceStateLabelValue      = "nice"
	softIRQStateLabelValue   = "softirq"
	stealStateLabelValue     = "steal"
	systemStateLabelValue    = "system"
	userStateLabelValue      = "user"
	waitStateLabelValue      = "wait"
)

// descriptors

var cpuTimeDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.InitEmpty()
	metric.SetName("system.cpu.time")
	metric.SetDescription("Total CPU seconds broken down by different states.")
	metric.SetUnit("s")
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	sum := metric.DoubleSum()
	sum.InitEmpty()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
