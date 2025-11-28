// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfighttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
)

func TestServerWithOtelHTTPOptions(t *testing.T) {
	// prepare
	sc := confighttp.ServerConfig{
		Endpoint: "localhost:0",
	}

	telemetry := componenttest.NewNopTelemetrySettings()
	tp, te := tracerProvider(t)
	telemetry.TracerProvider = tp

	srv, err := sc.ToServer(
		context.Background(),
		nil,
		telemetry,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		WithOtelHTTPOptions(
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return "example" + r.URL.Path
			}),
			otelhttp.WithFilter(func(r *http.Request) bool {
				return r.URL.Path != "/foobar"
			}),
		),
	)
	require.NoError(t, err)

	for _, path := range []string{"/path", "/foobar"} {
		response := &httptest.ResponseRecorder{}
		req, err := http.NewRequest(http.MethodGet, srv.Addr+path, http.NoBody)
		require.NoError(t, err)
		srv.Handler.ServeHTTP(response, req)
		assert.Equal(t, http.StatusOK, response.Result().StatusCode)
	}

	spans := te.GetSpans().Snapshots()
	assert.Len(t, spans, 1, "the request to /foobar should not be traced")
	assert.Equal(t, "example/path", spans[0].Name())
}

func tracerProvider(t *testing.T) (trace.TracerProvider, *tracetest.InMemoryExporter) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSyncer(exporter),
	)
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})
	return tp, exporter
}
