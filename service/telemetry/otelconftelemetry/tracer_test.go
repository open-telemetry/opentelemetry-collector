// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestCreateTracerProvider(t *testing.T) {
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

	resource, err := createResource(t.Context(), telemetry.Settings{
		BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
	}, cfg)
	require.NoError(t, err)

	provider, err := createTracerProvider(t.Context(), telemetry.TracerSettings{
		Settings: telemetry.Settings{
			BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
			Resource:  &resource,
		},
	}, cfg)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(t.Context()))
	}()

	tracer := provider.Tracer("test_tracer")
	_, span := tracer.Start(context.Background(), "test_span")
	span.End()

	require.Len(t, received, 1)
	traces := received[0]
	require.Equal(t, 1, traces.SpanCount())
	assert.Equal(t, "test_span", traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
}

func TestCreateTracerProvider_Invalid(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Logs.Level = zapcore.FatalLevel
	cfg.Metrics.Level = configtelemetry.LevelNone
	cfg.Traces.Processors = []config.SpanProcessor{{
		Simple: &config.SimpleSpanProcessor{
			Exporter: config.SpanExporter{
				OTLP: &config.OTLP{ /* missing endpoint, etc. */ },
			},
		},
	}}
	resource, err := createResource(t.Context(), telemetry.Settings{}, cfg)
	require.NoError(t, err)

	_, err = createTracerProvider(t.Context(), telemetry.TracerSettings{
		Settings: telemetry.Settings{Resource: &resource},
	}, cfg)
	require.EqualError(t, err, "no valid span exporter")
}

func TestCreateTracerProvider_Propagators(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", func(http.ResponseWriter, *http.Request) {})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := createDefaultConfig().(*Config)
	cfg.Traces.Propagators = []string{"b3", "tracecontext"}
	cfg.Traces.Processors = []config.SpanProcessor{newOTLPSimpleSpanProcessor(srv)}

	resource, err := createResource(t.Context(), telemetry.Settings{
		BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
	}, cfg)
	require.NoError(t, err)

	provider, err := createTracerProvider(t.Context(), telemetry.TracerSettings{
		Settings: telemetry.Settings{
			BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
			Resource:  &resource,
		},
	}, cfg)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(t.Context()))
	}()

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

func TestCreateTracerProvider_InvalidPropagator(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Traces.Propagators = []string{"invalid"}

	resource, err := createResource(t.Context(), telemetry.Settings{}, cfg)
	require.NoError(t, err)

	_, err = createTracerProvider(t.Context(), telemetry.TracerSettings{
		Settings: telemetry.Settings{Resource: &resource},
	}, cfg)
	assert.EqualError(t, err, "error creating propagator: unsupported trace propagator")
}

func TestCreateTracerProvider_Disabled(t *testing.T) {
	var received int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", func(http.ResponseWriter, *http.Request) {
		received++
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := createDefaultConfig().(*Config)
	cfg.Traces.Level = configtelemetry.LevelNone
	cfg.Traces.Processors = []config.SpanProcessor{newOTLPSimpleSpanProcessor(srv)}

	core, observedLogs := observer.New(zapcore.DebugLevel)

	resource, err := createResource(t.Context(), telemetry.Settings{
		BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
	}, cfg)
	require.NoError(t, err)

	settings := telemetry.TracerSettings{
		Settings: telemetry.Settings{
			BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
			Resource:  &resource,
		},
		Logger: zap.New(core),
	}

	provider, err := createTracerProvider(t.Context(), settings, cfg)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(t.Context()))
	}()

	require.Equal(t, 1, observedLogs.Len())
	assert.Equal(t, "Internal trace telemetry disabled", observedLogs.All()[0].Message)

	tracer := provider.Tracer("test_tracer")
	_, span := tracer.Start(context.Background(), "test_span")
	span.End()
	assert.Equal(t, 0, received)
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

// TestCreateTracerProvider_EnvVarDoesNotOverrideConfig verifies that the OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
// environment variable does not override the endpoint configured in the config file.
// This is a regression test for issue #14286.
func TestCreateTracerProvider_EnvVarDoesNotOverrideConfig(t *testing.T) {
	// Track which server receives requests
	var configuredReceived []ptrace.Traces
	var envVarReceived []ptrace.Traces

	// Create server for the configured endpoint
	configuredMux := http.NewServeMux()
	configuredMux.HandleFunc("/v1/traces", func(_ http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err)

		exportRequest := ptraceotlp.NewExportRequest()
		assert.NoError(t, exportRequest.UnmarshalProto(body))
		configuredReceived = append(configuredReceived, exportRequest.Traces())
	})
	configuredSrv := httptest.NewServer(configuredMux)
	defer configuredSrv.Close()

	// Create server for the env var endpoint (should not receive requests)
	envVarMux := http.NewServeMux()
	envVarMux.HandleFunc("/v1/traces", func(_ http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err)

		exportRequest := ptraceotlp.NewExportRequest()
		assert.NoError(t, exportRequest.UnmarshalProto(body))
		envVarReceived = append(envVarReceived, exportRequest.Traces())
	})
	envVarSrv := httptest.NewServer(envVarMux)
	defer envVarSrv.Close()

	// Set the environment variable to the env var endpoint
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", envVarSrv.URL+"/v1/traces")

	// Configure to use the configured endpoint (should not be overridden by env var)
	cfg := createDefaultConfig().(*Config)
	cfg.Traces.Propagators = []string{"tracecontext"}
	cfg.Traces.Processors = []config.SpanProcessor{
		{
			Simple: &config.SimpleSpanProcessor{
				Exporter: config.SpanExporter{
					OTLP: &config.OTLP{
						Endpoint: ptr(configuredSrv.URL + "/v1/traces"),
						Protocol: ptr("http/protobuf"),
						Insecure: ptr(true),
					},
				},
			},
		},
	}

	resource, err := createResource(t.Context(), telemetry.Settings{
		BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
	}, cfg)
	require.NoError(t, err)

	provider, err := createTracerProvider(t.Context(), telemetry.TracerSettings{
		Settings: telemetry.Settings{
			BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"},
			Resource:  &resource,
		},
	}, cfg)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, provider.Shutdown(t.Context()))
	}()

	tracer := provider.Tracer("test_tracer")
	_, span := tracer.Start(context.Background(), "test_span")
	span.End()

	// Verify that spans were sent to the configured endpoint
	// and NOT to the env var endpoint
	require.Len(t, configuredReceived, 1, "spans should be sent to the configured endpoint")
	require.Len(t, envVarReceived, 0, "spans should NOT be sent to the env var endpoint")

	traces := configuredReceived[0]
	require.Equal(t, 1, traces.SpanCount())
	assert.Equal(t, "test_span", traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
}

// TestUnsetAndRestoreEnvVars verifies that the environment variable
// unset/restore functions work correctly.
func TestUnsetAndRestoreEnvVars(t *testing.T) {
	// Set up some test environment variables
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://example.com")
	os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://traces.example.com")

	// Verify they are set
	_, exists := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	require.True(t, exists)
	_, exists = os.LookupEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	require.True(t, exists)

	// Unset them
	saved := unsetOTLPEndpointEnvVars()

	// Verify they are unset
	_, exists = os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	require.False(t, exists)
	_, exists = os.LookupEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	require.False(t, exists)

	// Restore them
	restoreEnvVars(saved)

	// Verify they are restored
	val, exists := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	require.True(t, exists)
	assert.Equal(t, "http://example.com", val)
	val, exists = os.LookupEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	require.True(t, exists)
	assert.Equal(t, "http://traces.example.com", val)

	// Clean up
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
}

