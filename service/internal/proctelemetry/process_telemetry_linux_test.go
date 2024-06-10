// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package proctelemetry

import (
	"fmt"
	"strings"
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTelemetryWithHostProc(t *testing.T) {
	tel := setupTelemetry(t)
	// Make the sure the environment variable value is not used.
	t.Setenv("HOST_PROC", "foo/bar")

	require.NoError(t, RegisterProcessMetrics(tel.TelemetrySettings, 0, WithHostProc("/proc")))

	// Check that the metrics are actually filled.
	time.Sleep(200 * time.Millisecond)

	mp, err := fetchPrometheusMetrics(tel.promHandler)
	require.NoError(t, err)

	for _, metricName := range expectedMetrics {
		metric, ok := mp[metricName]
		require.True(t, ok)
		require.True(t, len(metric.Metric) == 1)
		var metricValue float64
		if metric.GetType() == io_prometheus_client.MetricType_COUNTER {
			metricValue = metric.Metric[0].GetCounter().GetValue()
		} else {
			metricValue = metric.Metric[0].GetGauge().GetValue()
		}
		if strings.HasPrefix(metricName, "process_uptime") || strings.HasPrefix(metricName, "process_cpu_seconds") {
			// This likely will still be zero when running the test.
			assert.GreaterOrEqual(t, metricValue, float64(0), metricName)
			continue
		}

		fmt.Println(metricName)
		assert.Greater(t, metricValue, float64(0), metricName)
	}
}
