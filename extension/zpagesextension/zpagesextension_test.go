// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/internal/testutil"
)

type zpagesHost struct {
	component.Host
	mux *http.ServeMux
}

func newZPagesHost() *zpagesHost {
	return &zpagesHost{
		Host: componenttest.NewNopHost(),
		mux:  http.NewServeMux(),
	}
}

func (h *zpagesHost) GetZPagesMux() *http.ServeMux {
	return h.mux
}

func TestZPagesExtensionUsage(t *testing.T) {
	type testcase struct {
		expvarEnabled bool
	}

	for name, tc := range map[string]testcase{
		"expvar disabled": {
			expvarEnabled: false,
		},
		"expvar enabled": {
			expvarEnabled: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: testutil.GetAvailableLocalAddress(t),
				},
				Expvar: ExpvarConfig{
					Enabled: tc.expvarEnabled,
				},
			}

			zpagesExt := newServer(cfg, componenttest.NewNopTelemetrySettings())
			require.NotNil(t, zpagesExt)

			host := newZPagesHost()
			host.mux.HandleFunc("/debug/tracez", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			})
			require.NoError(t, zpagesExt.Start(t.Context(), host))
			t.Cleanup(func() {
				require.NoError(t, zpagesExt.Shutdown(context.WithoutCancel(t.Context())))
			})

			resp, err := http.Get("http://" + cfg.Endpoint + "/debug/tracez")
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusTeapot, resp.StatusCode)

			resp, err = http.Get("http://" + cfg.Endpoint + "/debug/expvarz")
			require.NoError(t, err)
			defer resp.Body.Close()
			if tc.expvarEnabled {
				require.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				require.Equal(t, http.StatusNotFound, resp.StatusCode)
			}
		})
	}
}

func TestZPagesMultipleStarts(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: testutil.GetAvailableLocalAddress(t),
		},
	}

	zpagesExt := newServer(cfg, componenttest.NewNopTelemetrySettings())
	require.NotNil(t, zpagesExt)

	host := newZPagesHost()
	require.NoError(t, zpagesExt.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, zpagesExt.Shutdown(context.Background())) })

	// Try to start it again, it will fail since it is on the same endpoint.
	require.Error(t, zpagesExt.Start(context.Background(), host))
}

func TestZPagesMultipleShutdowns(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: testutil.GetAvailableLocalAddress(t),
		},
	}

	zpagesExt := newServer(cfg, componenttest.NewNopTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, zpagesExt.Shutdown(context.Background()))
	require.NoError(t, zpagesExt.Shutdown(context.Background()))
}
