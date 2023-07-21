// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux
// +build linux

package proctelemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestOCProcessTelemetryWithHostProc(t *testing.T) {
	ocRegistry := metric.NewRegistry()
	// Make the sure the environment variable value is not used.
	t.Setenv("HOST_PROC", "foo/bar")

	require.NoError(t, RegisterProcessMetrics(ocRegistry, noop.NewMeterProvider(), false, 0, WithHostProc("/proc")))

	// Check that the metrics are actually filled.
	time.Sleep(200 * time.Millisecond)

	metrics := ocRegistry.Read()

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
			assert.GreaterOrEqual(t, value, float64(0), metricName)
			continue
		}
		assert.Greater(t, value, float64(0), metricName)
	}
}
