// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/testutil"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
)

func TestTelemetryConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		success bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
				},
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "127.0.0.1:3333",
				},
			},
			success: true,
		},
		{
			name: "Invalid config",
			cfg: &Config{
				Logs: LogsConfig{
					Level: zapcore.DebugLevel,
				},
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "127.0.0.1:3333",
				},
			},
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetry, err := New(context.Background(), Settings{ZapOptions: []zap.Option{}}, *tt.cfg)
			if tt.success {
				assert.NoError(t, err)
				assert.NotNil(t, telemetry)
			} else {
				assert.Error(t, err)
				assert.Nil(t, telemetry)
			}
		})
	}
}

func TestSampledLogger(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "Default sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Encoding: "console",
				},
			},
		},
		{
			name: "Custom sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
					Sampling: &LogsSamplingConfig{
						Enabled:    true,
						Tick:       1 * time.Second,
						Initial:    100,
						Thereafter: 100,
					},
				},
			},
		},
		{
			name: "Disable sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
					Sampling: &LogsSamplingConfig{
						Enabled: false,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetry, err := New(context.Background(), Settings{ZapOptions: []zap.Option{}}, *tt.cfg)
			assert.NoError(t, err)
			assert.NotNil(t, telemetry)
			assert.NotNil(t, telemetry.Logger())
		})
	}
}

const (
	metricPrefix = "otelcol_"
	otelPrefix   = "otel_sdk_"
	ocPrefix     = "oc_sdk_"
	grpcPrefix   = "gprc_"
	httpPrefix   = "http_"
	counterName  = "test_counter"
)

var testInstanceID = "test_instance_id"

func TestTelemetryInit(t *testing.T) {
	type metricValue struct {
		value  float64
		labels map[string]string
	}

	for _, tc := range []struct {
		name            string
		disableHighCard bool
		expectedMetrics map[string]metricValue
		extendedConfig  bool
		cfg             *Config
	}{
		{
			name: "UseOpenTelemetryForInternalMetrics",
			expectedMetrics: map[string]metricValue{
				metricPrefix + ocPrefix + counterName: {
					value: 13,
					labels: map[string]string{
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
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
						"net_sock_peer_addr":  "",
						"net_sock_peer_name":  "",
						"net_sock_peer_port":  "",
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
				metricPrefix + httpPrefix + counterName: {
					value: 10,
					labels: map[string]string{
						"net_host_name":       "",
						"net_host_port":       "",
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
			},
		},
		{
			name:            "DisableHighCardinalityWithOtel",
			disableHighCard: true,
			expectedMetrics: map[string]metricValue{
				metricPrefix + ocPrefix + counterName: {
					value: 13,
					labels: map[string]string{
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
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
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
				metricPrefix + httpPrefix + counterName: {
					value: 10,
					labels: map[string]string{
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
			},
		},
		{
			name:           "UseOTelWithSDKConfiguration",
			extendedConfig: true,
			cfg: &Config{
				Logs: LogsConfig{
					Encoding: "console",
				},
				Metrics: MetricsConfig{
					Level: configtelemetry.LevelDetailed,
				},
				Traces: TracesConfig{
					Processors: []config.SpanProcessor{
						{
							Batch: &config.BatchSpanProcessor{
								Exporter: config.SpanExporter{
									Console: config.Console{},
								},
							},
						},
					},
				},
				Resource: map[string]*string{
					semconv.AttributeServiceInstanceID: &testInstanceID,
				},
			},
			expectedMetrics: map[string]metricValue{
				metricPrefix + ocPrefix + counterName: {
					value: 13,
					labels: map[string]string{
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
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
						"net_sock_peer_addr":  "",
						"net_sock_peer_name":  "",
						"net_sock_peer_port":  "",
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
				metricPrefix + httpPrefix + counterName: {
					value: 10,
					labels: map[string]string{
						"net_host_name":       "",
						"net_host_port":       "",
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
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFactory(tc.disableHighCard, tc.extendedConfig)
			buildInfo := component.NewDefaultBuildInfo()
			if tc.extendedConfig {
				tc.cfg.Metrics.Readers = []config.MetricReader{
					{
						Pull: &config.PullMetricReader{
							Exporter: config.MetricExporter{
								Prometheus: testutil.GetAvailableLocalAddressPrometheus(t),
							},
						},
					},
				}
			}
			if tc.cfg == nil {
				tc.cfg = &Config{
					Resource: map[string]*string{
						semconv.AttributeServiceInstanceID: &testInstanceID,
					},
					Logs: LogsConfig{
						Encoding: "console",
					},
					Metrics: MetricsConfig{
						Level:   configtelemetry.LevelDetailed,
						Address: testutil.GetAvailableLocalAddress(t),
					},
				}
			}
			set := Settings{BuildInfo: buildInfo, AsyncErrorChannel: make(chan error)}
			tel, err := f.New(context.Background(), set, *tc.cfg)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tel.Shutdown(context.Background()))
			}()

			v := createTestMetrics(t, tel.MeterProvider())
			defer func() {
				view.Unregister(v)
			}()

			metrics := getMetricsFromPrometheus(t, tel.servers[0].Handler)
			require.Equal(t, len(tc.expectedMetrics), len(metrics))

			for metricName, metricValue := range tc.expectedMetrics {
				mf, present := metrics[metricName]
				require.True(t, present, "expected metric %q was not present", metricName)
				require.Len(t, mf.Metric, 1, "only one measure should exist for metric %q", metricName)

				labels := make(map[string]string)
				for _, pair := range mf.Metric[0].Label {
					labels[pair.GetName()] = pair.GetValue()
				}

				require.Equal(t, metricValue.labels, labels, "labels for metric %q was different than expected", metricName)
				require.Equal(t, metricValue.value, mf.Metric[0].Counter.GetValue(), "value for metric %q was different than expected", metricName)
			}
		})

	}
}

func createTestMetrics(t *testing.T, mp metric.MeterProvider) *view.View {
	// Creates a OTel Go counter
	counter, err := mp.Meter("collector_test").Int64Counter(otelPrefix+counterName, metric.WithUnit("ms"))
	require.NoError(t, err)
	counter.Add(context.Background(), 13)

	grpcExampleCounter, err := mp.Meter(proctelemetry.GRPCInstrumentation).Int64Counter(grpcPrefix + counterName)
	require.NoError(t, err)
	grpcExampleCounter.Add(context.Background(), 11, metric.WithAttributes(proctelemetry.GRPCUnacceptableKeyValues...))

	httpExampleCounter, err := mp.Meter(proctelemetry.HTTPInstrumentation).Int64Counter(httpPrefix + counterName)
	require.NoError(t, err)
	httpExampleCounter.Add(context.Background(), 10, metric.WithAttributes(proctelemetry.HTTPUnacceptableKeyValues...))

	// Creates a OpenCensus measure
	ocCounter := stats.Int64(ocPrefix+counterName, counterName, stats.UnitDimensionless)
	v := &view.View{
		Name:        ocPrefix + counterName,
		Description: ocCounter.Description(),
		Measure:     ocCounter,
		Aggregation: view.Sum(),
	}
	err = view.Register(v)
	require.NoError(t, err)

	stats.Record(context.Background(), stats.Int64(ocPrefix+counterName, counterName, stats.UnitDimensionless).M(13))

	// Forces a flush for the view data.
	_, _ = view.RetrieveData(ocPrefix + counterName)

	return v
}

func getMetricsFromPrometheus(t *testing.T, handler http.Handler) map[string]*io_prometheus_client.MetricFamily {
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(rr.Body)
	require.NoError(t, err)

	return parsed

}
