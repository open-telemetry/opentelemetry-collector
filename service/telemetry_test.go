// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/testutil"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	metricPrefix = "otelcol_"
	otelPrefix   = "otel_sdk_"
	ocPrefix     = "oc_sdk_"
	grpcPrefix   = "gprc_"
	httpPrefix   = "http_"
	counterName  = "test_counter"
)

func TestBuildResource(t *testing.T) {
	buildInfo := component.NewDefaultBuildInfo()

	// Check default config
	cfg := telemetry.Config{}
	otelRes := buildResource(buildInfo, cfg)
	res := pdataFromSdk(otelRes)

	assert.Equal(t, res.Attributes().Len(), 3)
	value, ok := res.Attributes().Get(semconv.AttributeServiceName)
	assert.True(t, ok)
	assert.Equal(t, buildInfo.Command, value.AsString())
	value, ok = res.Attributes().Get(semconv.AttributeServiceVersion)
	assert.True(t, ok)
	assert.Equal(t, buildInfo.Version, value.AsString())

	_, ok = res.Attributes().Get(semconv.AttributeServiceInstanceID)
	assert.True(t, ok)

	// Check override by nil
	cfg = telemetry.Config{
		Resource: map[string]*string{
			semconv.AttributeServiceName:       nil,
			semconv.AttributeServiceVersion:    nil,
			semconv.AttributeServiceInstanceID: nil,
		},
	}
	otelRes = buildResource(buildInfo, cfg)
	res = pdataFromSdk(otelRes)

	// Attributes should not exist since we nil-ified all.
	assert.Equal(t, res.Attributes().Len(), 0)

	// Check override values
	strPtr := func(v string) *string { return &v }
	cfg = telemetry.Config{
		Resource: map[string]*string{
			semconv.AttributeServiceName:       strPtr("a"),
			semconv.AttributeServiceVersion:    strPtr("b"),
			semconv.AttributeServiceInstanceID: strPtr("c"),
		},
	}
	otelRes = buildResource(buildInfo, cfg)
	res = pdataFromSdk(otelRes)

	assert.Equal(t, res.Attributes().Len(), 3)
	value, ok = res.Attributes().Get(semconv.AttributeServiceName)
	assert.True(t, ok)
	assert.Equal(t, "a", value.AsString())
	value, ok = res.Attributes().Get(semconv.AttributeServiceVersion)
	assert.True(t, ok)
	assert.Equal(t, "b", value.AsString())
	value, ok = res.Attributes().Get(semconv.AttributeServiceInstanceID)
	assert.True(t, ok)
	assert.Equal(t, "c", value.AsString())
}

func TestTelemetryInit(t *testing.T) {
	type metricValue struct {
		value  float64
		labels map[string]string
	}

	for _, tc := range []struct {
		name            string
		useOtel         bool
		disableHighCard bool
		expectedMetrics map[string]metricValue
		extendedConfig  bool
	}{
		{
			name:    "UseOpenCensusForInternalMetrics",
			useOtel: false,
			expectedMetrics: map[string]metricValue{
				metricPrefix + ocPrefix + counterName: {
					value: 13,
					labels: map[string]string{
						"service_name":        "otelcol",
						"service_version":     "latest",
						"service_instance_id": testInstanceID,
					},
				},
			},
		},
		{
			name:    "UseOpenTelemetryForInternalMetrics",
			useOtel: true,
			expectedMetrics: map[string]metricValue{
				metricPrefix + ocPrefix + counterName + "_total": {
					value:  13,
					labels: map[string]string{},
				},
				metricPrefix + otelPrefix + counterName + "_total": {
					value:  13,
					labels: map[string]string{},
				},
				metricPrefix + grpcPrefix + counterName + "_total": {
					value: 11,
					labels: map[string]string{
						"net_sock_peer_addr": "",
						"net_sock_peer_name": "",
						"net_sock_peer_port": "",
					},
				},
				metricPrefix + httpPrefix + counterName + "_total": {
					value: 10,
					labels: map[string]string{
						"net_host_name": "",
						"net_host_port": "",
					},
				},
				metricPrefix + "target_info": {
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
			useOtel:         true,
			disableHighCard: true,
			expectedMetrics: map[string]metricValue{
				metricPrefix + ocPrefix + counterName + "_total": {
					value:  13,
					labels: map[string]string{},
				},
				metricPrefix + otelPrefix + counterName + "_total": {
					value:  13,
					labels: map[string]string{},
				},
				metricPrefix + grpcPrefix + counterName + "_total": {
					value:  11,
					labels: map[string]string{},
				},
				metricPrefix + httpPrefix + counterName + "_total": {
					value:  10,
					labels: map[string]string{},
				},
				metricPrefix + "target_info": {
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
			tel := newColTelemetry(tc.useOtel, tc.disableHighCard, tc.extendedConfig)
			buildInfo := component.NewDefaultBuildInfo()
			cfg := telemetry.Config{
				Resource: map[string]*string{
					semconv.AttributeServiceInstanceID: &testInstanceID,
				},
				Metrics: telemetry.MetricsConfig{
					Level:   configtelemetry.LevelDetailed,
					Address: testutil.GetAvailableLocalAddress(t),
				},
			}
			otelRes := buildResource(buildInfo, cfg)
			res := pdataFromSdk(otelRes)
			settings := component.TelemetrySettings{
				Logger:   zap.NewNop(),
				Resource: res,
			}
			err := tel.init(otelRes, settings, cfg, make(chan error))
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tel.shutdown())
			}()

			v := createTestMetrics(t, tel.mp)
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

	grpcExampleCounter, err := mp.Meter(grpcInstrumentation).Int64Counter(grpcPrefix + counterName)
	require.NoError(t, err)
	grpcExampleCounter.Add(context.Background(), 11, metric.WithAttributes(grpcUnacceptableKeyValues...))

	httpExampleCounter, err := mp.Meter(httpInstrumentation).Int64Counter(httpPrefix + counterName)
	require.NoError(t, err)
	httpExampleCounter.Add(context.Background(), 10, metric.WithAttributes(httpUnacceptableKeyValues...))

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
