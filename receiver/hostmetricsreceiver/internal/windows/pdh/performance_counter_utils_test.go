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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/third_party/telegraf/win_perf_counters"
)

func TestPerfCounter_InitializeMetric_NoLabels(t *testing.T) {
	data := []win_perf_counters.CounterValue{{InstanceName: "_Total", Value: 100}}

	metric := dataold.NewMetric()
	metric.InitEmpty()
	InitializeMetric(metric, data, "")

	ddp := metric.DoubleDataPoints()
	assert.Equal(t, 1, ddp.Len())
	assert.Equal(t, 0, ddp.At(0).LabelsMap().Len())
	assert.Equal(t, float64(100), ddp.At(0).Value())
}

func TestPerfCounter_InitializeMetric_Labels(t *testing.T) {
	data := []win_perf_counters.CounterValue{{InstanceName: "label_value_1", Value: 20}, {InstanceName: "label_value_2", Value: 50}}

	metric := dataold.NewMetric()
	metric.InitEmpty()
	InitializeMetric(metric, data, "label")

	ddp := metric.DoubleDataPoints()
	assert.Equal(t, 2, ddp.Len())
	assert.Equal(t, pdata.NewStringMap().InitFromMap(map[string]string{"label": "label_value_1"}), ddp.At(0).LabelsMap().Sort())
	assert.Equal(t, float64(20), ddp.At(0).Value())
	assert.Equal(t, pdata.NewStringMap().InitFromMap(map[string]string{"label": "label_value_2"}), ddp.At(1).LabelsMap().Sort())
	assert.Equal(t, float64(50), ddp.At(1).Value())
}
