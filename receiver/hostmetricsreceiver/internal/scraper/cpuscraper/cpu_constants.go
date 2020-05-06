// Copyright 2020, OpenTelemetry Authors
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

package cpuscraper

import (
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// cpu metric constants

const (
	stateLabelName = "state"
	cpuLabelName   = "cpu"
)

const (
	userStateLabelValue      = "user"
	systemStateLabelValue    = "system"
	idleStateLabelValue      = "idle"
	interruptStateLabelValue = "interrupt"
	niceStateLabelValue      = "nice"
	softIRQStateLabelValue   = "softirq"
	stealStateLabelValue     = "steal"
	waitStateLabelValue      = "wait"
)

var metricCPUSecondsDescriptor = createMetricCPUSecondsDescriptor()

func createMetricCPUSecondsDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/cpu/time")
	descriptor.SetDescription("Total CPU ticks or jiffies broken down by different states")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeCounterInt64)
	return descriptor
}
