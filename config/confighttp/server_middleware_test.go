// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

// testServerMiddleware is a test implementation of configmiddleware.Config
type testServerMiddleware struct {
	extension.Extension
	extensionmiddleware.GetHTTPHandlerFunc
}

func newTestServerMiddleware(name string) component.Component {
	return &testServerMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetHTTPHandlerFunc: func(_ context.Context) (extensionmiddleware.WrapHTTPHandlerFunc, error) {
			return func(_ context.Context, handler http.Handler) (http.Handler, error) {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Append middleware name to the URL path
					r.URL.Path += name + "/"

					// Call the next handler in the chain
					handler.ServeHTTP(w, r)

					// Add middleware name to the response
					_, _ = w.Write([]byte("\r\nserved by " + name))
				}), nil
			}, nil
		},
	}
}

func newTestServerConfig(name string) configmiddleware.Config {
	return configmiddleware.Config{
		ID: component.MustNewID(name),
	}
}

// testServerProtocolsMiddleware adds HTTPServerProtocols to a test middleware.
type testServerProtocolsMiddleware struct {
	component.Component
	extensionmiddleware.GetHTTPServerProtocolsFunc
}

func (m *testServerProtocolsMiddleware) GetHTTPHandler(ctx context.Context) (extensionmiddleware.WrapHTTPHandlerFunc, error) {
	return m.Component.(extensionmiddleware.HTTPServer).GetHTTPHandler(ctx)
}

func TestServerMiddleware(t *testing.T) {
	// Register two test extensions
	extensions := map[component.ID]component.Component{
		component.MustNewID("test1"): newTestServerMiddleware("test1"),
		component.MustNewID("test2"): newTestServerMiddleware("test2"),
	}

	// Test with different middleware configurations
	testCases := []struct {
		name           string
		middlewares    []configmiddleware.Config
		expectedOutput string
	}{
		{
			name:           "no_middlewares",
			middlewares:    nil,
			expectedOutput: "OK{/}",
		},
		{
			name: "single_middleware",
			middlewares: []configmiddleware.Config{
				newTestServerConfig("test1"),
			},
			expectedOutput: "OK{/test1/}\r\nserved by test1",
		},
		{
			name: "multiple_middlewares",
			middlewares: []configmiddleware.Config{
				newTestServerConfig("test1"),
				newTestServerConfig("test2"),
			},
			expectedOutput: "OK{/test1/test2/}\r\nserved by test2\r\nserved by test1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create server config with the test middlewares
			cfg := ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: tc.middlewares,
			}

			// Create a test handler that responds with the request path
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("OK{" + r.URL.Path + "}")) // #nosec G705
			})

			// Create the server
			srv, err := cfg.ToServer(
				context.Background(),
				extensions,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.NoError(t, err)

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

			// Create a response recorder
			rec := httptest.NewRecorder()

			// Serve the request
			srv.Handler.ServeHTTP(rec, req)

			// Get the response
			resp := rec.Result()
			defer resp.Body.Close()

			// Check the response
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedOutput, string(body))
		})
	}
}

func TestServerMiddlewareProtocols(t *testing.T) {
	unencryptedHTTP2 := func(context.Context) (http.Protocols, error) {
		var p http.Protocols
		p.SetUnencryptedHTTP2(true)
		return p, nil
	}
	extensions := map[component.ID]component.Component{
		component.MustNewID("plain"): newTestServerMiddleware("plain"),
		component.MustNewID("h2c"): &testServerProtocolsMiddleware{
			Component:                  newTestServerMiddleware("h2c"),
			GetHTTPServerProtocolsFunc: unencryptedHTTP2,
		},
		component.MustNewID("defaults"): &testServerProtocolsMiddleware{
			Component: newTestServerMiddleware("defaults"),
			GetHTTPServerProtocolsFunc: func(context.Context) (http.Protocols, error) {
				var p http.Protocols
				p.SetHTTP1(true)
				p.SetHTTP2(true)
				return p, nil
			},
		},
		component.MustNewID("broken"): &testServerProtocolsMiddleware{
			Component: newTestServerMiddleware("broken"),
			GetHTTPServerProtocolsFunc: func(context.Context) (http.Protocols, error) {
				return http.Protocols{}, errors.New("protocols error")
			},
		},
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	start := func(t *testing.T, tlsConfig configoptional.Optional[configtls.ServerConfig], middlewares ...string) (*http.Server, string) {
		t.Helper()
		cfg := ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  "localhost:0",
				Transport: confignet.TransportTypeTCP,
			},
			TLS: tlsConfig,
		}
		for _, name := range middlewares {
			cfg.Middlewares = append(cfg.Middlewares, newTestServerConfig(name))
		}
		srv, err := cfg.ToServer(t.Context(), extensions, componenttest.NewNopTelemetrySettings(), handler)
		require.NoError(t, err)
		ln, err := cfg.ToListener(t.Context())
		require.NoError(t, err)
		go func() { _ = srv.Serve(ln) }()
		t.Cleanup(func() { require.NoError(t, srv.Close()) })
		return srv, ln.Addr().String()
	}
	// get returns the HTTP major version the client negotiated.
	get := func(addr string, protocols func(*http.Protocols)) (int, error) {
		var p http.Protocols
		protocols(&p)
		client := &http.Client{Transport: &http.Transport{Protocols: &p}, Timeout: 5 * time.Second}
		defer client.CloseIdleConnections()
		resp, err := client.Get("http://" + addr)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		return resp.ProtoMajor, nil
	}
	http1 := func(p *http.Protocols) { p.SetHTTP1(true) }
	h2c := func(p *http.Protocols) { p.SetUnencryptedHTTP2(true) }
	noTLS := configoptional.None[configtls.ServerConfig]()

	t.Run("unchanged_without_request", func(t *testing.T) {
		srv, addr := start(t, noTLS, "plain")
		assert.Nil(t, srv.Protocols)
		_, err := get(addr, h2c)
		require.Error(t, err)
	})

	t.Run("unchanged_when_requesting_defaults", func(t *testing.T) {
		srv, addr := start(t, noTLS, "defaults")
		assert.Nil(t, srv.Protocols)
		_, err := get(addr, h2c)
		require.Error(t, err)
	})

	t.Run("adds_requested_protocols", func(t *testing.T) {
		srv, addr := start(t, noTLS, "plain", "h2c")
		require.NotNil(t, srv.Protocols)
		assert.True(t, srv.Protocols.HTTP1())
		assert.True(t, srv.Protocols.HTTP2())
		assert.True(t, srv.Protocols.UnencryptedHTTP2())

		major, err := get(addr, h2c)
		require.NoError(t, err)
		assert.Equal(t, 2, major)

		major, err = get(addr, http1)
		require.NoError(t, err)
		assert.Equal(t, 1, major)
	})

	t.Run("tls_without_cleartext_requirement", func(t *testing.T) {
		serverTLS := configtls.ServerConfig{
			Config: configtls.Config{
				CertFile: filepath.Join("testdata", "server.crt"),
				KeyFile:  filepath.Join("testdata", "server.key"),
			},
		}
		_, addr := start(t, configoptional.Some(serverTLS), "defaults")

		_, err := get(addr, h2c)
		require.Error(t, err, "TLS listener must reject prior-knowledge h2c")
		require.NotErrorIs(t, err, context.DeadlineExceeded)

		cc := NewDefaultClientConfig()
		cc.Timeout = 5 * time.Second
		cc.ForceAttemptHTTP2 = true
		cc.TLS.CAFile = filepath.Join("testdata", "ca.crt")
		cc.TLS.ServerName = "localhost"
		client, err := cc.ToClient(t.Context(), nil, nilProvidersSettings)
		require.NoError(t, err)
		t.Cleanup(client.CloseIdleConnections)

		resp, err := client.Get("https://" + addr)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		assert.NotNil(t, resp.TLS)
		assert.Equal(t, 2, resp.ProtoMajor)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("tls_rejects_cleartext_requirement", func(t *testing.T) {
		id := component.MustNewIDWithName("protocols", "proxy")
		cfg := ServerConfig{
			TLS:         configoptional.Some(configtls.ServerConfig{}),
			Middlewares: []configmiddleware.Config{{ID: id}},
		}
		srv, err := cfg.ToServer(t.Context(), map[component.ID]component.Component{id: extensions[component.MustNewID("h2c")]}, componenttest.NewNopTelemetrySettings(), handler)
		require.Nil(t, srv)
		require.ErrorContains(t, err, `middleware "protocols/proxy" requests unencrypted HTTP/2, but this listener requires TLS`)
	})

	t.Run("returns_error", func(t *testing.T) {
		cfg := ServerConfig{Middlewares: []configmiddleware.Config{newTestServerConfig("broken")}}
		_, err := cfg.ToServer(t.Context(), extensions, componenttest.NewNopTelemetrySettings(), handler)
		require.ErrorContains(t, err, "protocols error")
	})
}

func TestServerMiddlewareErrors(t *testing.T) {
	// Create a basic handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	// Test cases for HTTP server middleware errors
	httpTests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ServerConfig
		errText    string
	}{
		{
			name:       "extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_http_handler_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("http middleware error")),
			},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "http middleware error",
		},
	}

	for _, tc := range httpTests {
		t.Run(tc.name, func(t *testing.T) {
			// Trying to create the server should fail
			_, err := tc.config.ToServer(
				context.Background(),
				tc.extensions,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}

	// Test cases for gRPC server middleware errors
	grpcTests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ServerConfig
		errText    string
	}{
		{
			name:       "grpc_extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_grpc_handler_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("grpc middleware error")),
			},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "grpc middleware error",
		},
	}

	for _, tc := range grpcTests {
		t.Run(tc.name, func(t *testing.T) {
			// Trying to create the server should fail
			_, err := tc.config.ToServer(
				context.Background(),
				tc.extensions,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}
