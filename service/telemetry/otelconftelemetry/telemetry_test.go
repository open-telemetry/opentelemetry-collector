// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestCreateProviders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(http.ResponseWriter, *http.Request) {})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Logs.Level = zapcore.FatalLevel
	cfg.Metrics.Level = configtelemetry.LevelBasic
	cfg.Metrics.Readers = []config.MetricReader{{
		Periodic: &config.PeriodicMetricReader{
			Exporter: config.PushMetricExporter{
				OTLP: &config.OTLPMetric{
					Endpoint: ptr(srv.URL),
					Protocol: ptr("http/protobuf"),
				},
			},
		},
	}}
	cfg.Traces.Level = configtelemetry.LevelBasic
	cfg.Traces.Processors = []config.SpanProcessor{{
		Batch: &config.BatchSpanProcessor{
			Exporter: config.SpanExporter{
				OTLP: &config.OTLP{
					Endpoint: ptr(srv.URL),
					Protocol: ptr("http/protobuf"),
				},
			},
		},
	}}

	providers, err := factory.CreateProviders(t.Context(), telemetry.Settings{}, cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	defer func() {
		require.NoError(t, providers.Shutdown(t.Context()))
	}()

	assert.NotZero(t, providers.Resource())
	assert.NotNil(t, providers.Logger())
	assert.NotNil(t, providers.LoggerProvider())
	assert.NotNil(t, providers.MeterProvider())
	assert.NotNil(t, providers.TracerProvider())
}
