// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

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
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/config/configtelemetry"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	"go.opentelemetry.io/collector/service/internal/promtest"
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
		disableHighCard bool
		expectedMetrics map[string]metricValue
	}{
		{
			name: "UseOpenTelemetryForInternalMetrics",
			expectedMetrics: map[string]metricValue{
				metricPrefix + otelPrefix + counterName: {
					value: 13,
					labels: map[string]string{
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
					},
				},
				metricPrefix + grpcPrefix + counterName: {
					value: 11,
					labels: map[string]string{
						"net_sock_peer_addr":     "",
						"net_sock_peer_name":     "",
						"net_sock_peer_port":     "",
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
					},
				},
				metricPrefix + httpPrefix + counterName: {
					value: 10,
					labels: map[string]string{
						"net_host_name":          "",
						"net_host_port":          "",
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
					},
				},
				"target_info": {
					value: 0,
					labels: map[string]string{
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
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
		{
			name:            "DisableHighCardinalityWithOtel",
			disableHighCard: true,
			expectedMetrics: map[string]metricValue{
				metricPrefix + otelPrefix + counterName: {
					value: 13,
					labels: map[string]string{
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
					},
				},
				metricPrefix + grpcPrefix + counterName: {
					value: 11,
					labels: map[string]string{
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
					},
				},
				metricPrefix + httpPrefix + counterName: {
					value: 10,
					labels: map[string]string{
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
					},
				},
				"target_info": {
					value: 0,
					labels: map[string]string{
						"service_name":           "otelcol",
						"service_version":        "latest",
						"service_instance_id":    testInstanceID,
						"telemetry_sdk_language": "go",
						"telemetry_sdk_name":     "opentelemetry",
						"telemetry_sdk_version":  "1.31.0",
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
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Metrics: MetricsConfig{
					Level: configtelemetry.LevelDetailed,
					Readers: []config.MetricReader{{
						Pull: &config.PullMetricReader{Exporter: config.PullMetricExporter{Prometheus: prom}},
					}},
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
					semconv.AttributeServiceName:       ptr("otelcol"),
					semconv.AttributeServiceVersion:    ptr("latest"),
				},
			}
			mp, err := newMeterProvider(context.Background(), Settings{}, *cfg, tt.disableHighCard)
			require.NoError(t, err)
			defer func() {
				if prov, ok := mp.(interface{ Shutdown(context.Context) error }); ok {
					require.NoError(t, prov.Shutdown(context.Background()))
				}
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
		attribute.String(semconv.AttributeNetSockPeerAddr, ""),
		attribute.String(semconv.AttributeNetSockPeerPort, ""),
		attribute.String(semconv.AttributeNetSockPeerName, ""),
	)))

	httpExampleCounter, err := mp.Meter("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp").Int64Counter(metricPrefix + httpPrefix + counterName)
	require.NoError(t, err)
	httpExampleCounter.Add(context.Background(), 10, metric.WithAttributeSet(attribute.NewSet(
		attribute.String(semconv.AttributeNetHostName, ""),
		attribute.String(semconv.AttributeNetHostPort, ""),
	)))
}

func getMetricsFromPrometheus(t *testing.T, endpoint string) map[string]*io_prometheus_client.MetricFamily {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
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

// Test that the MeterProvider implements the 'Enabled' functionality.
// See https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric/internal/x#readme-instrument-enabled.
func TestInstrumentEnabled(t *testing.T) {
	prom := promtest.GetAvailableLocalAddressPrometheus(t)
	set := meterProviderSettings{
		res: sdkresource.Default(),
		cfg: MetricsConfig{
			Level: configtelemetry.LevelDetailed,
			Readers: []config.MetricReader{{
				Pull: &config.PullMetricReader{Exporter: config.MetricExporter{Prometheus: prom}},
			}},
		},
		asyncErrorChannel: make(chan error),
	}
	meterProvider, err := newMeterProvider(set, false)
	defer func() {
		if prov, ok := meterProvider.(interface{ Shutdown(context.Context) error }); ok {
			require.NoError(t, prov.Shutdown(context.Background()))
		}
	}()
	require.NoError(t, err)

	meter := meterProvider.Meter("go.opentelemetry.io/collector/service/telemetry")

	type enabledInstrument interface{ Enabled(context.Context) bool }

	intCnt, err := meter.Int64Counter("int64.counter")
	require.NoError(t, err)
	_, ok := intCnt.(enabledInstrument)
	assert.True(t, ok, "Int64Counter does not implement the experimental 'Enabled' method")

	intUpDownCnt, err := meter.Int64UpDownCounter("int64.updowncounter")
	require.NoError(t, err)
	_, ok = intUpDownCnt.(enabledInstrument)
	assert.True(t, ok, "Int64UpDownCounter does not implement the experimental 'Enabled' method")

	intHist, err := meter.Int64Histogram("int64.updowncounter")
	require.NoError(t, err)
	_, ok = intHist.(enabledInstrument)
	assert.True(t, ok, "Int64Histogram does not implement the experimental 'Enabled' method")

	intGauge, err := meter.Int64Gauge("int64.updowncounter")
	require.NoError(t, err)
	_, ok = intGauge.(enabledInstrument)
	assert.True(t, ok, "Int64Gauge does not implement the experimental 'Enabled' method")

	floatCnt, err := meter.Float64Counter("int64.updowncounter")
	require.NoError(t, err)
	_, ok = floatCnt.(enabledInstrument)
	assert.True(t, ok, "Float64Counter does not implement the experimental 'Enabled' method")

	floatUpDownCnt, err := meter.Float64UpDownCounter("int64.updowncounter")
	require.NoError(t, err)
	_, ok = floatUpDownCnt.(enabledInstrument)
	assert.True(t, ok, "Float64UpDownCounter does not implement the experimental 'Enabled' method")

	floatHist, err := meter.Float64Histogram("int64.updowncounter")
	require.NoError(t, err)
	_, ok = floatHist.(enabledInstrument)
	assert.True(t, ok, "Float64Histogram does not implement the experimental 'Enabled' method")

	floatGauge, err := meter.Float64Gauge("int64.updowncounter")
	require.NoError(t, err)
	_, ok = floatGauge.(enabledInstrument)
	assert.True(t, ok, "Float64Gauge does not implement the experimental 'Enabled' method")
}
