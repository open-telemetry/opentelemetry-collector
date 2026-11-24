// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"
)

// TestServerMetricsIgnoreHostHeader verifies that the server.address and
// server.port attributes of the HTTP server metrics come from the configured
// endpoint and not from the Host header of the request, which is chosen by the
// peer, and is recorded even when authentication rejects the request.
func TestServerMetricsIgnoreHostHeader(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { assert.NoError(t, mp.Shutdown(t.Context())) })

	telemetry := componenttest.NewNopTelemetrySettings()
	telemetry.MeterProvider = mp

	authID := component.MustNewID("testauth")
	sc := NewDefaultServerConfig()
	sc.NetAddr = confignet.AddrConfig{Endpoint: "configured.example:4318", Transport: confignet.TransportTypeTCP}
	sc.Auth = configoptional.Some(AuthConfig{Config: configauth.Config{AuthenticatorID: authID}})

	extensions := map[component.ID]component.Component{
		authID: extensionauthtest.NewErr(errors.New("authentication failed")),
	}
	srv, err := sc.ToServer(t.Context(), extensions, telemetry, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.NoError(t, err)

	hosts := []string{"attacker-chosen.example", strings.Repeat("a", 1<<16) + ":9999"}
	for _, host := range hosts {
		req := httptest.NewRequest(http.MethodPost, "/v1/traces", http.NoBody)
		req.Host = host
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		// The request is rejected by authentication, and still recorded in the metrics.
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	var seen int
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if !strings.HasPrefix(m.Name, "http.server.") {
				continue
			}
			seen++
			for _, set := range attributeSets(t, m) {
				addr, ok := set.Value("server.address")
				require.True(t, ok, "metric %s has no server.address", m.Name)
				assert.Equal(t, "configured.example", addr.AsString(), "metric %s", m.Name)
				port, ok := set.Value("server.port")
				if ok {
					assert.Equal(t, int64(4318), port.AsInt64(), "metric %s", m.Name)
				}
			}
		}
	}
	assert.NotZero(t, seen, "no http.server.* metrics were recorded")
}

func attributeSets(t *testing.T, m metricdata.Metrics) []attribute.Set {
	t.Helper()
	var sets []attribute.Set
	switch data := m.Data.(type) {
	case metricdata.Histogram[int64]:
		for _, dp := range data.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, dp := range data.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	case metricdata.Sum[int64]:
		for _, dp := range data.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	default:
		t.Fatalf("unexpected data type %T for metric %s", m.Data, m.Name)
	}
	return sets
}
