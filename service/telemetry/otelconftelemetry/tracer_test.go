// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestTracerProvider(t *testing.T) {
	var received []ptrace.Traces
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", func(_ http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err)

		exportRequest := ptraceotlp.NewExportRequest()
		assert.NoError(t, exportRequest.UnmarshalProto(body))
		received = append(received, exportRequest.Traces())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := createDefaultConfig().(*Config)
	cfg.Traces.Propagators = []string{"b3", "tracecontext"}
	cfg.Traces.Processors = []config.SpanProcessor{newOTLPSimpleSpanProcessor(srv)}

	buildInfo := component.BuildInfo{Command: "otelcol", Version: "latest"}
	providers, _ := newTelemetryProviders(t, telemetry.Settings{BuildInfo: buildInfo}, cfg)

	provider := providers.TracerProvider()
	tracer := provider.Tracer("test_tracer")
	_, span := tracer.Start(context.Background(), "test_span")
	span.End()

	require.Len(t, received, 1)
	traces := received[0]
	require.Equal(t, 1, traces.SpanCount())
	assert.Equal(t, "test_span", traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
}

func TestTelemetry_TracerProvider_Propagators(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", func(http.ResponseWriter, *http.Request) {})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := createDefaultConfig().(*Config)
	cfg.Traces.Propagators = []string{"b3", "tracecontext"}
	cfg.Traces.Processors = []config.SpanProcessor{newOTLPSimpleSpanProcessor(srv)}

	buildInfo := component.BuildInfo{Command: "otelcol", Version: "latest"}
	providers, _ := newTelemetryProviders(t, telemetry.Settings{BuildInfo: buildInfo}, cfg)

	provider := providers.TracerProvider()
	propagator := otel.GetTextMapPropagator()
	require.NotNil(t, propagator)

	tracer := provider.Tracer("test_tracer")
	ctx, span := tracer.Start(context.Background(), "test_span")
	mapCarrier := make(propagation.MapCarrier)
	propagator.Inject(ctx, mapCarrier)
	span.End()

	assert.Contains(t, mapCarrier, "b3")
	assert.Contains(t, mapCarrier, "traceparent")
}

func TestTelemetry_TracerProviderDisabled(t *testing.T) {
	test := func(t *testing.T, cfg *Config) {
		t.Helper()

		var received int
		mux := http.NewServeMux()
		mux.HandleFunc("/v1/traces", func(http.ResponseWriter, *http.Request) {
			received++
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		cfg.Traces.Processors = []config.SpanProcessor{
			newOTLPSimpleSpanProcessor(srv),
		}

		buildInfo := component.BuildInfo{Command: "otelcol", Version: "latest"}
		providers, _ := newTelemetryProviders(t, telemetry.Settings{BuildInfo: buildInfo}, cfg)

		provider := providers.TracerProvider()
		tracer := provider.Tracer("test_tracer")
		_, span := tracer.Start(context.Background(), "test_span")
		span.End()
		assert.Equal(t, 0, received)
	}

	t.Run("level_none", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Traces.Level = configtelemetry.LevelNone
		test(t, cfg)
	})
}

func newOTLPSimpleSpanProcessor(srv *httptest.Server) config.SpanProcessor {
	return config.SpanProcessor{
		Simple: &config.SimpleSpanProcessor{
			Exporter: config.SpanExporter{
				OTLP: &config.OTLP{
					Endpoint: ptr(srv.URL),
					Protocol: ptr("http/protobuf"),
					Insecure: ptr(true),
				},
			},
		},
	}
}
