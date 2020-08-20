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

// +build windows

package pdh

import (
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/third_party/telegraf/win_perf_counters"
)

// InitializeMetric initializes the provided metric with
// datapoints from the specified Counter Values.
//
// The performance counters' "instance" will be recorded
// against the supplied label name
func InitializeMetric(
	metric pdata.Metric,
	vals []win_perf_counters.CounterValue,
	instanceNameLabel string,
) pdata.Metric {
	ddps := metric.DoubleDataPoints()
	ddps.Resize(len(vals))

	for i, val := range vals {
		ddp := ddps.At(i)

		if len(vals) > 1 || (val.InstanceName != "" && val.InstanceName != totalInstanceName) {
			labels := ddp.LabelsMap()
			labels.Insert(instanceNameLabel, val.InstanceName)
		}

		ddp.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
		ddp.SetValue(val.Value)
	}

	return metric
}
