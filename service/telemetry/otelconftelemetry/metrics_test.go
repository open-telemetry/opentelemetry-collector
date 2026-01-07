// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/service/internal/promtest"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	metricPrefix = "otelcol_"
	otelPrefix   = "otel_sdk_"
	grpcPrefix   = "grpc_"
	httpPrefix   = "http_"
	counterName  = "test_counter"
)

var testInstanceID = "test_instance_id"

func TestCreateMeterProvider(t *testing.T) {
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

		cfg := createDefaultConfig().(*Config)
		cfg.Metrics = MetricsConfig{
			Level: configtelemetry.LevelDetailed,
			MeterProvider: config.MeterProvider{
				Readers: []config.MetricReader{{
					Pull: &config.PullMetricReader{
						Exporter: config.PullMetricExporter{Prometheus: prom},
					},
				}},
			},
		}
		cfg.Resource = map[string]*string{
			"service.name":        ptr("otelcol"),
			"service.version":     ptr("latest"),
			"service.instance.id": ptr(testInstanceID),
		}

		t.Run(tt.name, func(t *testing.T) {
			resource, err := createResource(t.Context(), telemetry.Settings{}, cfg)
			require.NoError(t, err)

			mp, err := createMeterProvider(t.Context(), telemetry.MeterSettings{
				Settings: telemetry.Settings{Resource: &resource},
			}, cfg)
			require.NoError(t, err)
			defer func() {
				assert.NoError(t, mp.Shutdown(t.Context()))
			}()

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
		attribute.String("rpc.system", "grpc"),
	)))

	httpExampleCounter, err := mp.Meter("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp").Int64Counter(metricPrefix + httpPrefix + counterName)
	require.NoError(t, err)
	httpExampleCounter.Add(context.Background(), 10, metric.WithAttributeSet(attribute.NewSet(
		attribute.String("http.request.method", "GET"),
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
	for i := range maxRetries {
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

	parser := expfmt.NewTextParser(model.UTF8Validation)
	parsed, err := parser.TextToMetricFamilies(rr.Body)
	require.NoError(t, err)

	return parsed
}

func TestCreateMeterProvider_Invalid(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Logs.Level = zapcore.FatalLevel
	cfg.Traces.Level = configtelemetry.LevelNone
	cfg.Metrics.Readers = []config.MetricReader{{
		// Invalid -- no OTLP protocol defined
		Periodic: &config.PeriodicMetricReader{Exporter: config.PushMetricExporter{OTLP: &config.OTLPMetric{}}},
	}}
	resource, err := createResource(t.Context(), telemetry.Settings{}, cfg)
	require.NoError(t, err)

	_, err = createMeterProvider(t.Context(), telemetry.MeterSettings{
		Settings: telemetry.Settings{Resource: &resource},
	}, cfg)
	require.EqualError(t, err, "no valid metric exporter")
}

func TestCreateMeterProvider_Disabled(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Metrics.Readers = []config.MetricReader{{
		// Invalid -- no OTLP protocol defined
		Periodic: &config.PeriodicMetricReader{Exporter: config.PushMetricExporter{OTLP: &config.OTLPMetric{}}},
	}}

	core, observedLogs := observer.New(zapcore.DebugLevel)

	factory := NewFactory()
	resource, err := factory.CreateResource(context.Background(), telemetry.Settings{}, cfg)
	require.NoError(t, err)

	settings := telemetry.MeterSettings{
		Settings: telemetry.Settings{Resource: &resource},
	}
	settings.Logger = zap.New(core)

	_, err = factory.CreateMeterProvider(context.Background(), settings, cfg)
	require.EqualError(t, err, "no valid metric exporter")
	assert.Zero(t, observedLogs.Len())

	// Setting Metrics.Level to LevelNone disables metrics,
	// so the invalid configuration should not cause an error.
	cfg.Metrics.Level = configtelemetry.LevelNone
	resource2, err := createResource(t.Context(), telemetry.Settings{}, cfg)
	require.NoError(t, err)
	settings.Resource = &resource2

	mp, err := createMeterProvider(t.Context(), settings, cfg)
	require.NoError(t, err)
	assert.NoError(t, mp.Shutdown(t.Context()))

	require.Equal(t, 1, observedLogs.Len())
	assert.Equal(t, "Internal metrics telemetry disabled", observedLogs.All()[0].Message)
}

// Test that the MeterProvider implements the 'Enabled' functionality.
// See https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric/internal/x#readme-instrument-enabled.
func TestInstrumentEnabled(t *testing.T) {
	prom := promtest.GetAvailableLocalAddressPrometheus(t)
	cfg := createDefaultConfig().(*Config)
	cfg.Metrics.Readers = []config.MetricReader{{
		Pull: &config.PullMetricReader{Exporter: config.PullMetricExporter{Prometheus: prom}},
	}}

	resource, err := createResource(t.Context(), telemetry.Settings{}, cfg)
	require.NoError(t, err)

	meterProvider, err := createMeterProvider(t.Context(), telemetry.MeterSettings{
		Settings: telemetry.Settings{Resource: &resource},
	}, cfg)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, meterProvider.Shutdown(t.Context()))
	}()

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

func TestTelemetryMetrics_DefaultViews(t *testing.T) {
	type testcase struct {
		configuredViews    []config.View
		expectedMeterNames []string
	}

	defaultViews := func(level configtelemetry.Level) []config.View {
		assert.Equal(t, configtelemetry.LevelDetailed, level)
		return []config.View{{
			Selector: &config.ViewSelector{
				MeterName: ptr("a"),
			},
			Stream: &config.ViewStream{
				Aggregation: &config.ViewStreamAggregation{
					Drop: config.ViewStreamAggregationDrop{},
				},
			},
		}}
	}

	for name, tc := range map[string]testcase{
		"no_configured_views": {
			expectedMeterNames: []string{"b", "c"},
		},
		"configured_views": {
			configuredViews: []config.View{{
				Selector: &config.ViewSelector{
					MeterName: ptr("b"),
				},
				Stream: &config.ViewStream{
					Aggregation: &config.ViewStreamAggregation{
						Drop: config.ViewStreamAggregationDrop{},
					},
				},
			}},
			expectedMeterNames: []string{"a", "c"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var metrics pmetric.Metrics
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)

				exportRequest := pmetricotlp.NewExportRequest()
				assert.NoError(t, exportRequest.UnmarshalProto(body))
				metrics = exportRequest.Metrics()
			}))
			defer srv.Close()

			cfg := createDefaultConfig().(*Config)
			cfg.Metrics.Level = configtelemetry.LevelDetailed
			cfg.Metrics.Views = tc.configuredViews
			cfg.Metrics.Readers = []config.MetricReader{{
				Periodic: &config.PeriodicMetricReader{
					Exporter: config.PushMetricExporter{
						OTLP: &config.OTLPMetric{
							Endpoint: ptr(srv.URL),
							Protocol: ptr("http/protobuf"),
							Insecure: ptr(true),
						},
					},
				},
			}}

			factory := NewFactory()
			settings := telemetry.MeterSettings{DefaultViews: defaultViews}
			provider, err := factory.CreateMeterProvider(t.Context(), settings, cfg)
			require.NoError(t, err)

			for _, meterName := range []string{"a", "b", "c"} {
				counter, _ := provider.Meter(meterName).Int64Counter(meterName + ".counter")
				counter.Add(t.Context(), 1)
			}
			require.NoError(t, provider.Shutdown(t.Context())) // should flush metrics

			var scopes []string
			for _, rm := range metrics.ResourceMetrics().All() {
				for _, sm := range rm.ScopeMetrics().All() {
					scopes = append(scopes, sm.Scope().Name())
				}
			}
			assert.ElementsMatch(t, tc.expectedMeterNames, scopes)
		})
	}
}
