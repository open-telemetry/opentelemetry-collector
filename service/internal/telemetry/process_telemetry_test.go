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

package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
)

var expectedMetrics = []string{
	// Changing a metric name is a breaking change.
	// Adding new metrics is ok as long it follows the conventions described at
	// https://pkg.go.dev/go.opentelemetry.io/collector/obsreport?tab=doc#hdr-Naming_Convention_for_New_Metrics
	"process/uptime",
	"process/runtime/heap_alloc_bytes",
	"process/runtime/total_alloc_bytes",
	"process/runtime/total_sys_memory_bytes",
	"process/cpu_seconds",
	"process/memory/rss",
}

func TestProcessTelemetry(t *testing.T) {
	registry := metric.NewRegistry()
	require.NoError(t, RegisterProcessMetrics(registry, 0))

	// Check that the metrics are actually filled.
	<-time.After(200 * time.Millisecond)

	metrics := registry.Read()

	for _, metricName := range expectedMetrics {
		m := findMetric(metrics, metricName)
		require.NotNil(t, m)
		require.Len(t, m.TimeSeries, 1)
		ts := m.TimeSeries[0]
		assert.Len(t, ts.LabelValues, 0)
		require.Len(t, ts.Points, 1)

		var value float64
		if metricName == "process/uptime" || metricName == "process/cpu_seconds" {
			value = ts.Points[0].Value.(float64)
		} else {
			value = float64(ts.Points[0].Value.(int64))
		}

		if metricName == "process/uptime" || metricName == "process/cpu_seconds" {
			// This likely will still be zero when running the test.
			assert.True(t, value >= 0, metricName)
			continue
		}

		assert.True(t, value > 0, metricName)
	}
}

func TestProcessTelemetryFailToRegister(t *testing.T) {

	for _, metricName := range expectedMetrics {
		t.Run(metricName, func(t *testing.T) {
			registry := metric.NewRegistry()
			_, err := registry.AddFloat64Gauge(metricName)
			require.NoError(t, err)
			assert.Error(t, RegisterProcessMetrics(registry, 0))
		})
	}
}

func findMetric(metrics []*metricdata.Metric, name string) *metricdata.Metric {
	for _, m := range metrics {
		if m.Descriptor.Name == name {
			return m
		}
	}
	return nil
}
