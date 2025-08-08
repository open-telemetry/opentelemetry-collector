// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/internal/promtest"
	"go.opentelemetry.io/collector/service/internal/resource"
)

const (
	metricPrefix = "otelcol_"
	otelPrefix   = "otel_sdk_"
	grpcPrefix   = "grpc_"
	httpPrefix   = "http_"
	counterName  = "test_counter"
)

var testInstanceID = "test_instance_id"

func TestTelemetryInit(t *testing.T) {
	type metricValue struct {
		value  float64
		labels map[string]string
	}

	for _, tt := range []struct {
		name            string
		expectedMetrics map[string]metricValue
	}{
		{
			name: "UseOpenTelemetryForInternalMetrics",
			expectedMetrics: map[string]metricValue{
				metricPrefix + otelPrefix + counterName: {
					value: 13,
					labels: map[string]string{
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
				metricPrefix + grpcPrefix + counterName: {
					value: 11,
					labels: map[string]string{
						"rpc_system":          "grpc",
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
				metricPrefix + httpPrefix + counterName: {
					value: 10,
					labels: map[string]string{
						"http_request_method": "GET",
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
				"target_info": {
					value: 0,
					labels: map[string]string{
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
				"promhttp_metric_handler_errors_total": {
					value: 0,
					labels: map[string]string{
						"cause": "encoding",
					},
				},
			},
		},
	} {
		prom := promtest.GetAvailableLocalAddressPrometheus(t)
		endpoint := fmt.Sprintf("http://%s:%d/metrics", *prom.Host, *prom.Port)
		cfg := Config{
			Metrics: MetricsConfig{
				Level: configtelemetry.LevelDetailed,
				MeterProvider: config.MeterProvider{
					Readers: []config.MetricReader{{
						Pull: &config.PullMetricReader{
							Exporter: config.PullMetricExporter{Prometheus: prom},
						},
					}},
				},
			},
		}
		t.Run(tt.name, func(t *testing.T) {
			res := resource.New(component.BuildInfo{}, map[string]*string{
				string(semconv.ServiceNameKey):       ptr("otelcol"),
				string(semconv.ServiceVersionKey):    ptr("latest"),
				string(semconv.ServiceInstanceIDKey): ptr(testInstanceID),
			})
			sdk, err := NewSDK(context.Background(), &cfg, res)
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, sdk.Shutdown(context.Background()))
			})

			mp, err := newMeterProvider(cfg, sdk)
			require.NoError(t, err)

			createTestMetrics(t, mp)

			metrics := getMetricsFromPrometheus(t, endpoint)
			require.Len(t, metrics, len(tt.expectedMetrics))

			for metricName, metricValue := range tt.expectedMetrics {
				mf, present := metrics[metricName]
				require.True(t, present, "expected metric %q was not present", metricName)
				if metricName == "promhttp_metric_handler_errors_total" {
					continue
				}
				require.Len(t, mf.Metric, 1, "only one measure should exist for metric %q", metricName)

				labels := make(map[string]string)
				for _, pair := range mf.Metric[0].Label {
					labels[pair.GetName()] = pair.GetValue()
				}

				require.Equal(t, metricValue.labels, labels, "labels for metric %q was different than expected", metricName)
				require.InDelta(t, metricValue.value, mf.Metric[0].Counter.GetValue(), 0.01, "value for metric %q was different than expected", metricName)
			}
		})
	}
}

func createTestMetrics(t *testing.T, mp metric.MeterProvider) {
	// Creates a OTel Go counter
	counter, err := mp.Meter("collector_test").Int64Counter(metricPrefix+otelPrefix+counterName, metric.WithUnit("ms"))
	require.NoError(t, err)
	counter.Add(context.Background(), 13)

	grpcExampleCounter, err := mp.Meter("go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc").Int64Counter(metricPrefix + grpcPrefix + counterName)
	require.NoError(t, err)
	grpcExampleCounter.Add(context.Background(), 11, metric.WithAttributeSet(attribute.NewSet(
		attribute.String(string(semconv.RPCSystemKey), "grpc"),
	)))

	httpExampleCounter, err := mp.Meter("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp").Int64Counter(metricPrefix + httpPrefix + counterName)
	require.NoError(t, err)
	httpExampleCounter.Add(context.Background(), 10, metric.WithAttributeSet(attribute.NewSet(
		attribute.String(string(semconv.HTTPRequestMethodKey), "GET"),
	)))
}

func getMetricsFromPrometheus(t *testing.T, endpoint string) map[string]*io_prometheus_client.MetricFamily {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, http.NoBody)
	require.NoError(t, err)

	var rr *http.Response
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		rr, err = client.Do(req)
		if err == nil && rr.StatusCode == http.StatusOK {
			break
		}

		// skip sleep on last retry
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second) // Wait before retrying
		}
	}
	require.NoError(t, err, "failed to get metrics from Prometheus after %d attempts", maxRetries)
	require.Equal(t, http.StatusOK, rr.StatusCode, "unexpected status code after %d attempts", maxRetries)
	defer rr.Body.Close()

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(rr.Body)
	require.NoError(t, err)

	return parsed
}

func TestTelemetryMetricsDisabled(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Metrics.Readers = []config.MetricReader{{
		// Invalid -- no OTLP protocol defined
		Periodic: &config.PeriodicMetricReader{Exporter: config.PushMetricExporter{OTLP: &config.OTLPMetric{}}},
	}}

	res := resource.New(component.BuildInfo{}, nil)
	_, err := NewSDK(context.Background(), cfg, res)
	require.EqualError(t, err, "no valid metric exporter")

	// Setting Metrics.Level to LevelNone disables metrics,
	// so the invalid configuration should not cause an error.
	cfg.Metrics.Level = configtelemetry.LevelNone
	sdk, err := NewSDK(context.Background(), cfg, res)
	require.NoError(t, err)
	assert.NoError(t, sdk.Shutdown(context.Background()))
}

// Test that the MeterProvider implements the 'Enabled' functionality.
// See https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric/internal/x#readme-instrument-enabled.
func TestInstrumentEnabled(t *testing.T) {
	prom := promtest.GetAvailableLocalAddressPrometheus(t)
	cfg := createDefaultConfig().(*Config)
	cfg.Metrics.Readers = []config.MetricReader{{
		Pull: &config.PullMetricReader{Exporter: config.PullMetricExporter{Prometheus: prom}},
	}}

	sdk, err := NewSDK(context.Background(), cfg, resource.New(component.BuildInfo{}, nil))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sdk.Shutdown(context.Background()))
	})
	require.NoError(t, err)

	meterProvider, err := newMeterProvider(*cfg, sdk)
	require.NoError(t, err)

	meter := meterProvider.Meter("go.opentelemetry.io/collector/service/telemetry")

	type enabledInstrument interface{ Enabled(context.Context) bool }

	intCnt, err := meter.Int64Counter("int64.counter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), intCnt)

	intUpDownCnt, err := meter.Int64UpDownCounter("int64.updowncounter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), intUpDownCnt)

	intHist, err := meter.Int64Histogram("int64.updowncounter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), intHist)

	intGauge, err := meter.Int64Gauge("int64.updowncounter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), intGauge)

	floatCnt, err := meter.Float64Counter("int64.updowncounter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), floatCnt)

	floatUpDownCnt, err := meter.Float64UpDownCounter("int64.updowncounter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), floatUpDownCnt)

	floatHist, err := meter.Float64Histogram("int64.updowncounter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), floatHist)

	floatGauge, err := meter.Float64Gauge("int64.updowncounter")
	require.NoError(t, err)
	assert.Implements(t, new(enabledInstrument), floatGauge)
}
