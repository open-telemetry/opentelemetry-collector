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

package loadscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// descriptors

var loadAvg1MDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.cpu.load_average.1m")
	metric.SetDescription("Average CPU Load over 1 minute.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
	return metric
}()

var loadAvg5mDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.cpu.load_average.5m")
	metric.SetDescription("Average CPU Load over 5 minutes.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
	return metric
}()

var loadAvg15mDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.SetName("system.cpu.load_average.15m")
	metric.SetDescription("Average CPU Load over 15 minutes.")
	metric.SetUnit("1")
	metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
	return metric
}()
