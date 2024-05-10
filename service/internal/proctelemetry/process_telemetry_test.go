// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

type testTelemetry struct {
	component.TelemetrySettings
	promHandler   http.Handler
	meterProvider *sdkmetric.MeterProvider
}

var expectedMetrics = []string{
	"process_uptime",
	"process_runtime_heap_alloc_bytes",
	"process_runtime_total_alloc_bytes",
	"process_runtime_total_sys_memory_bytes",
	"process_cpu_seconds",
	"process_memory_rss",
}

func setupTelemetry(t *testing.T) testTelemetry {
	settings := testTelemetry{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}
	settings.TelemetrySettings.MetricsLevel = configtelemetry.LevelNormal

	promReg := prometheus.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(promReg), otelprom.WithoutUnits(), otelprom.WithoutCounterSuffixes())
	require.NoError(t, err)

	settings.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.Empty()),
		sdkmetric.WithReader(exporter),
	)
	settings.TelemetrySettings.MeterProvider = settings.meterProvider

	settings.promHandler = promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})

	t.Cleanup(func() { assert.NoError(t, settings.meterProvider.Shutdown(context.Background())) })

	return settings
}

func fetchPrometheusMetrics(handler http.Handler) (map[string]*io_prometheus_client.MetricFamily, error) {
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	if err != nil {
		return nil, err
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(rr.Body)
}

func TestProcessTelemetry(t *testing.T) {
	tel := setupTelemetry(t)

	require.NoError(t, RegisterProcessMetrics(tel.MeterProvider, 0))

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

		assert.Greater(t, metricValue, float64(0), metricName)
	}
}
