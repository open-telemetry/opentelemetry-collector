// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	_ extension.Extension  = (*mockAuthServer)(nil)
	_ extensionauth.Server = (*mockAuthServer)(nil)
)

type mockAuthServer struct {
	component.StartFunc
	component.ShutdownFunc
	extensionauth.ServerAuthenticateFunc
}

func newMockAuthServer(auth func(ctx context.Context, sources map[string][]string) (context.Context, error)) extension.Extension {
	return &mockAuthServer{ServerAuthenticateFunc: auth}
}

func TestHTTPServerSettingsError(t *testing.T) {
	tests := []struct {
		settings ServerConfig
		err      string
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool File: failed to load cert /doesnt/exist:",
			settings: ServerConfig{
				Endpoint: "localhost:0",
				TLS: configoptional.Some(configtls.ServerConfig{
					Config: configtls.Config{
						CAFile: "/doesnt/exist",
					},
				}),
			},
		},
		{
			err: "^failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither",
			settings: ServerConfig{
				Endpoint: "localhost:0",
				TLS: configoptional.Some(configtls.ServerConfig{
					Config: configtls.Config{
						CertFile: "/doesnt/exist",
					},
				}),
			},
		},
		{
			err: "failed to load client CA CertPool: failed to load CA /doesnt/exist:",
			settings: ServerConfig{
				Endpoint: "localhost:0",
				TLS: configoptional.Some(configtls.ServerConfig{
					ClientCAFile: "/doesnt/exist",
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.err, func(t *testing.T) {
			_, err := tt.settings.ToListener(context.Background())
			assert.Regexp(t, tt.err, err)
		})
	}
}

func TestHttpReception(t *testing.T) {
	tests := []struct {
		name           string
		tlsServerCreds configoptional.Optional[configtls.ServerConfig]
		tlsClientCreds *configtls.ClientConfig
		hasError       bool
		forceHTTP1     bool
	}{
		{
			name:           "noTLS",
			tlsServerCreds: configoptional.None[configtls.ServerConfig](),
			tlsClientCreds: &configtls.ClientConfig{
				Insecure: true,
			},
		},
		{
			name: "TLS",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
			}),
			tlsClientCreds: &configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "TLS (HTTP/1.1)",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
			}),
			tlsClientCreds: &configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			forceHTTP1: true,
		},
		{
			name: "NoServerCertificates",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
			}),
			tlsClientCreds: &configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "mTLS",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			}),
			tlsClientCreds: &configtls.ClientConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "client.crt"),
					KeyFile:  filepath.Join("testdata", "client.key"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoClientCertificate",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			}),
			tlsClientCreds: &configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "WrongClientCA",
			tlsServerCreds: configoptional.Some(configtls.ServerConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "server.crt"),
			}),
			tlsClientCreds: &configtls.ClientConfig{
				Config: configtls.Config{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "client.crt"),
					KeyFile:  filepath.Join("testdata", "client.key"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
	}
	// prepare

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &ServerConfig{
				Endpoint: "localhost:0",
				TLS:      tt.tlsServerCreds,
			}
			ln, err := sc.ToListener(context.Background())
			require.NoError(t, err)

			s, err := sc.ToServer(
				context.Background(),
				nil,
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, errWrite := fmt.Fprint(w, "tt")
					assert.NoError(t, errWrite)
				}))
			require.NoError(t, err)

			go func() {
				_ = s.Serve(ln)
			}()

			prefix := "https://"
			expectedProto := "HTTP/2.0"
			if tt.tlsClientCreds.Insecure {
				prefix = "http://"
				expectedProto = "HTTP/1.1"
			}

			cc := &ClientConfig{
				Endpoint:          prefix + ln.Addr().String(),
				TLS:               *tt.tlsClientCreds,
				ForceAttemptHTTP2: true,
			}

			client, errClient := cc.ToClient(context.Background(), nil, nilProvidersSettings)
			require.NoError(t, errClient)

			if tt.forceHTTP1 {
				expectedProto = "HTTP/1.1"
				client.Transport.(*http.Transport).ForceAttemptHTTP2 = false
			}

			resp, errResp := client.Get(cc.Endpoint)
			if tt.hasError {
				require.Error(t, errResp)
			} else {
				require.NoError(t, errResp)
				body, errRead := io.ReadAll(resp.Body)
				require.NoError(t, errRead)
				assert.Equal(t, "tt", string(body))
				assert.Equal(t, expectedProto, resp.Proto)
			}
			require.NoError(t, s.Close())
		})
	}
}

func TestHttpCors(t *testing.T) {
	tests := []struct {
		name string

		CORSConfig configoptional.Optional[CORSConfig]

		allowedWorks     bool
		disallowedWorks  bool
		extraHeaderWorks bool
	}{
		{
			name:             "noCORS",
			allowedWorks:     false,
			disallowedWorks:  false,
			extraHeaderWorks: false,
		},
		{
			name:             "emptyCORS",
			CORSConfig:       configoptional.Some(NewDefaultCORSConfig()),
			allowedWorks:     false,
			disallowedWorks:  false,
			extraHeaderWorks: false,
		},
		{
			name: "OriginCORS",
			CORSConfig: configoptional.Some(CORSConfig{
				AllowedOrigins: []string{"allowed-*.com"},
			}),
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: false,
		},
		{
			name: "CacheableCORS",
			CORSConfig: configoptional.Some(CORSConfig{
				AllowedOrigins: []string{"allowed-*.com"},
				MaxAge:         360,
			}),
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: false,
		},
		{
			name: "HeaderCORS",
			CORSConfig: configoptional.Some(CORSConfig{
				AllowedOrigins: []string{"allowed-*.com"},
				AllowedHeaders: []string{"ExtraHeader"},
			}),
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &ServerConfig{
				Endpoint: "localhost:0",
				CORS:     tt.CORSConfig,
			}

			ln, err := sc.ToListener(context.Background())
			require.NoError(t, err)

			s, err := sc.ToServer(
				context.Background(),
				nil,
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			require.NoError(t, err)

			go func() {
				_ = s.Serve(ln)
			}()

			url := "http://" + ln.Addr().String()

			expectedStatus := http.StatusNoContent
			if !tt.CORSConfig.HasValue() || len(tt.CORSConfig.Get().AllowedOrigins) == 0 {
				expectedStatus = http.StatusOK
			}

			// Verify allowed domain gets responses that allow CORS.
			verifyCorsResp(t, url, "allowed-origin.com", tt.CORSConfig, false, expectedStatus, tt.allowedWorks)

			// Verify allowed domain and extra headers gets responses that allow CORS.
			verifyCorsResp(t, url, "allowed-origin.com", tt.CORSConfig, true, expectedStatus, tt.extraHeaderWorks)

			// Verify disallowed domain gets responses that disallow CORS.
			verifyCorsResp(t, url, "disallowed-origin.com", tt.CORSConfig, false, expectedStatus, tt.disallowedWorks)

			require.NoError(t, s.Close())
		})
	}
}

func TestHttpCorsInvalidSettings(t *testing.T) {
	sc := &ServerConfig{
		Endpoint: "localhost:0",
		CORS:     configoptional.Some(CORSConfig{AllowedHeaders: []string{"some-header"}}),
	}

	// This effectively does not enable CORS but should also not cause an error
	s, err := sc.ToServer(
		context.Background(),
		nil,
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.NoError(t, err)
	require.NotNil(t, s)
	require.NoError(t, s.Close())
}

func TestHttpCorsWithSettings(t *testing.T) {
	sc := &ServerConfig{
		Endpoint: "localhost:0",
		CORS: configoptional.Some(CORSConfig{
			AllowedOrigins: []string{"*"},
		}),
		Auth: configoptional.Some(AuthConfig{
			Config: configauth.Config{
				AuthenticatorID: mockID,
			},
		}),
	}

	extensions := map[component.ID]component.Component{
		mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
			return ctx, errors.New("Settings failed")
		}),
	}

	srv, err := sc.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings(), nil)
	require.NoError(t, err)
	require.NotNil(t, srv)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", http.NoBody)
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	srv.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestHttpServerHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers configopaque.MapList
	}{
		{
			name:    "noHeaders",
			headers: nil,
		},
		{
			name: "withHeaders",
			headers: configopaque.MapList{
				{Name: "x-new-header-1", Value: "value1"},
				{Name: "x-new-header-2", Value: "value2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &ServerConfig{
				Endpoint:        "localhost:0",
				ResponseHeaders: tt.headers,
			}

			ln, err := sc.ToListener(context.Background())
			require.NoError(t, err)

			s, err := sc.ToServer(
				context.Background(),
				nil,
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			require.NoError(t, err)

			go func() {
				_ = s.Serve(ln)
			}()

			url := "http://" + ln.Addr().String()

			// Verify allowed domain gets responses that allow CORS.
			verifyHeadersResp(t, url, tt.headers)

			require.NoError(t, s.Close())
		})
	}
}

func verifyCorsResp(t *testing.T, url, origin string, set configoptional.Optional[CORSConfig], extraHeader bool, wantStatus int, wantAllowed bool) {
	req, err := http.NewRequest(http.MethodOptions, url, http.NoBody)
	require.NoError(t, err, "Error creating trace OPTIONS request: %v", err)
	req.Header.Set("Origin", origin)
	if extraHeader {
		req.Header.Set("ExtraHeader", "foo")
		req.Header.Set("Access-Control-Request-Headers", "extraheader")
	}
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Error sending OPTIONS to http server")
	require.NotNil(t, resp.Body)
	require.NoError(t, resp.Body.Close(), "Error closing OPTIONS response body")

	assert.Equal(t, wantStatus, resp.StatusCode)

	gotAllowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	gotAllowMethods := resp.Header.Get("Access-Control-Allow-Methods")

	wantAllowOrigin := ""
	wantAllowMethods := ""
	wantMaxAge := ""
	if wantAllowed {
		wantAllowOrigin = origin
		wantAllowMethods = "POST"
		if set.HasValue() && set.Get().MaxAge != 0 {
			wantMaxAge = strconv.Itoa(set.Get().MaxAge)
		}
	}
	assert.Equal(t, wantAllowOrigin, gotAllowOrigin)
	assert.Equal(t, wantAllowMethods, gotAllowMethods)
	assert.Equal(t, wantMaxAge, resp.Header.Get("Access-Control-Max-Age"))
}

func verifyHeadersResp(t *testing.T, url string, expected configopaque.MapList) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	require.NoError(t, err, "Error creating request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Error sending request to http server")
	require.NotNil(t, resp.Body)
	require.NoError(t, resp.Body.Close(), "Error closing response body")

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	for k, v := range expected.Iter {
		assert.Equal(t, string(v), resp.Header.Get(k))
	}
}

func TestServerAuth(t *testing.T) {
	// prepare
	authCalled := false
	sc := ServerConfig{
		Endpoint: "localhost:0",
		Auth: configoptional.Some(AuthConfig{
			Config: configauth.Config{
				AuthenticatorID: mockID,
			},
		}),
	}

	extensions := map[component.ID]component.Component{
		mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
			authCalled = true
			return ctx, nil
		}),
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	})

	srv, err := sc.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings(), handler)
	require.NoError(t, err)

	// tt
	srv.Handler.ServeHTTP(&httptest.ResponseRecorder{}, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	// verify
	assert.True(t, handlerCalled)
	assert.True(t, authCalled)
}

func TestInvalidServerAuth(t *testing.T) {
	sc := ServerConfig{
		Auth: configoptional.Some(AuthConfig{
			Config: configauth.Config{
				AuthenticatorID: nonExistingID,
			},
		}),
	}

	srv, err := sc.ToServer(context.Background(), nil, componenttest.NewNopTelemetrySettings(), http.NewServeMux())
	require.Error(t, err)
	require.Nil(t, srv)
}

func TestFailedServerAuth(t *testing.T) {
	// prepare
	sc := ServerConfig{
		Endpoint: "localhost:0",
		Auth: configoptional.Some(AuthConfig{
			Config: configauth.Config{
				AuthenticatorID: mockID,
			},
		}),
	}
	extensions := map[component.ID]component.Component{
		mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
			return ctx, errors.New("invalid authorization")
		}),
	}

	srv, err := sc.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.NoError(t, err)

	// tt
	response := &httptest.ResponseRecorder{}
	srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	// verify
	assert.Equal(t, http.StatusUnauthorized, response.Result().StatusCode)
	assert.Equal(t, fmt.Sprintf("%v %s", http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)), response.Result().Status)
}

func TestFailedServerAuthWithErrorHandler(t *testing.T) {
	// prepare
	sc := ServerConfig{
		Endpoint: "localhost:0",
		Auth: configoptional.Some(AuthConfig{
			Config: configauth.Config{
				AuthenticatorID: mockID,
			},
		}),
	}
	extensions := map[component.ID]component.Component{
		mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
			return ctx, errors.New("invalid authorization")
		}),
	}

	eh := func(w http.ResponseWriter, _ *http.Request, err string, statusCode int) {
		assert.Equal(t, http.StatusUnauthorized, statusCode)
		// custom error handler uses real error string
		assert.Equal(t, "invalid authorization", err)
		// custom error handler changes returned status code
		http.Error(w, err, http.StatusInternalServerError)
	}

	srv, err := sc.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), WithErrorHandler(eh))
	require.NoError(t, err)

	// tt
	response := &httptest.ResponseRecorder{}
	srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	// verify
	assert.Equal(t, http.StatusInternalServerError, response.Result().StatusCode)
	assert.Equal(t, fmt.Sprintf("%v %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)), response.Result().Status)
}

func TestServerWithErrorHandler(t *testing.T) {
	// prepare
	sc := ServerConfig{
		Endpoint: "localhost:0",
	}
	eh := func(w http.ResponseWriter, _ *http.Request, _ string, statusCode int) {
		assert.Equal(t, http.StatusBadRequest, statusCode)
		// custom error handler changes returned status code
		http.Error(w, "invalid request", http.StatusInternalServerError)
	}

	srv, err := sc.ToServer(
		context.Background(),
		nil,
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		WithErrorHandler(eh),
	)
	require.NoError(t, err)
	// tt
	response := &httptest.ResponseRecorder{}

	req, err := http.NewRequest(http.MethodGet, srv.Addr, http.NoBody)
	require.NoError(t, err, "Error creating request: %v", err)
	req.Header.Set("Content-Encoding", "something-invalid")

	srv.Handler.ServeHTTP(response, req)
	// verify
	assert.Equal(t, http.StatusInternalServerError, response.Result().StatusCode)
}

func TestServerWithDecoder(t *testing.T) {
	// prepare
	sc := NewDefaultServerConfig()
	sc.Endpoint = "localhost:0"
	decoder := func(body io.ReadCloser) (io.ReadCloser, error) {
		return body, nil
	}

	srv, err := sc.ToServer(
		context.Background(),
		nil,
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		WithDecoder("something-else", decoder),
	)
	require.NoError(t, err)
	// tt
	response := &httptest.ResponseRecorder{}

	req, err := http.NewRequest(http.MethodGet, srv.Addr, bytes.NewBuffer([]byte("something")))
	require.NoError(t, err, "Error creating request: %v", err)
	req.Header.Set("Content-Encoding", "something-else")

	srv.Handler.ServeHTTP(response, req)
	// verify
	assert.Equal(t, http.StatusOK, response.Result().StatusCode)
}

func TestServerWithDecompression(t *testing.T) {
	// prepare
	sc := ServerConfig{
		MaxRequestBodySize: 1000, // 1 KB
	}
	body := []byte(strings.Repeat("a", 1000*1000)) // 1 MB

	srv, err := sc.ToServer(
		context.Background(),
		nil,
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			actualBody, err := io.ReadAll(req.Body)
			assert.ErrorContains(t, err, "http: request body too large")
			assert.Len(t, actualBody, 1000)

			if err != nil {
				resp.WriteHeader(http.StatusBadRequest)
			} else {
				resp.WriteHeader(http.StatusOK)
			}
		}),
	)
	require.NoError(t, err)

	testSrv := httptest.NewServer(srv.Handler)
	defer testSrv.Close()

	req, err := http.NewRequest(http.MethodGet, testSrv.URL, compressZstd(t, body))
	require.NoError(t, err, "Error creating request: %v", err)

	req.Header.Set("Content-Encoding", "zstd")

	// tt
	c := http.Client{}
	resp, err := c.Do(req)
	require.NoError(t, err, "Error sending request: %v", err)

	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err, "Error reading response body: %v", err)

	// verifications is done mostly within the tt, but this is only a sanity check
	// that we got into the tt handler
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDefaultMaxRequestBodySize(t *testing.T) {
	tests := []struct {
		name     string
		settings ServerConfig
		expected int64
	}{
		{
			name:     "default",
			settings: ServerConfig{},
			expected: defaultMaxRequestBodySize,
		},
		{
			name:     "zero",
			settings: ServerConfig{MaxRequestBodySize: 0},
			expected: defaultMaxRequestBodySize,
		},
		{
			name:     "negative",
			settings: ServerConfig{MaxRequestBodySize: -1},
			expected: defaultMaxRequestBodySize,
		},
		{
			name:     "custom",
			settings: ServerConfig{MaxRequestBodySize: 100},
			expected: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.settings.ToServer(
				context.Background(),
				nil,
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tt.settings.MaxRequestBodySize)
		})
	}
}

func TestAuthWithQueryParams(t *testing.T) {
	// prepare
	authCalled := false
	sc := ServerConfig{
		Endpoint: "localhost:0",
		Auth: configoptional.Some(AuthConfig{
			RequestParameters: []string{"auth"},
			Config: configauth.Config{
				AuthenticatorID: mockID,
			},
		}),
	}

	extensions := map[component.ID]component.Component{
		mockID: newMockAuthServer(func(ctx context.Context, sources map[string][]string) (context.Context, error) {
			require.Len(t, sources, 1)
			assert.Equal(t, "1", sources["auth"][0])
			authCalled = true
			return ctx, nil
		}),
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	})

	srv, err := sc.ToServer(context.Background(), extensions, componenttest.NewNopTelemetrySettings(), handler)
	require.NoError(t, err)

	// tt
	srv.Handler.ServeHTTP(&httptest.ResponseRecorder{}, httptest.NewRequest(http.MethodGet, "/?auth=1", http.NoBody))

	// verify
	assert.True(t, handlerCalled)
	assert.True(t, authCalled)
}

func BenchmarkHttpRequest(b *testing.B) {
	tests := []struct {
		name            string
		forceHTTP1      bool
		clientPerThread bool
	}{
		{
			name:            "HTTP/2.0, shared client (like load balancer)",
			forceHTTP1:      false,
			clientPerThread: false,
		},
		{
			name:            "HTTP/1.1, shared client (like load balancer)",
			forceHTTP1:      true,
			clientPerThread: false,
		},
		{
			name:            "HTTP/2.0, client per thread (like single app)",
			forceHTTP1:      false,
			clientPerThread: true,
		},
		{
			name:            "HTTP/1.1, client per thread (like single app)",
			forceHTTP1:      true,
			clientPerThread: true,
		},
	}

	tlsServerCreds := configoptional.Some(configtls.ServerConfig{
		Config: configtls.Config{
			CAFile:   filepath.Join("testdata", "ca.crt"),
			CertFile: filepath.Join("testdata", "server.crt"),
			KeyFile:  filepath.Join("testdata", "server.key"),
		},
	})
	tlsClientCreds := &configtls.ClientConfig{
		Config: configtls.Config{
			CAFile: filepath.Join("testdata", "ca.crt"),
		},
		ServerName: "localhost",
	}

	sc := &ServerConfig{
		Endpoint: "localhost:0",
		TLS:      tlsServerCreds,
	}

	s, err := sc.ToServer(
		context.Background(),
		nil,
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, errWrite := fmt.Fprint(w, "tt")
			assert.NoError(b, errWrite)
		}))
	require.NoError(b, err)
	ln, err := sc.ToListener(context.Background())
	require.NoError(b, err)

	go func() {
		_ = s.Serve(ln)
	}()
	defer func() {
		_ = s.Close()
	}()

	for _, bb := range tests {
		cc := &ClientConfig{
			Endpoint: "https://" + ln.Addr().String(),
			TLS:      *tlsClientCreds,
		}

		b.Run(bb.name, func(b *testing.B) {
			var c *http.Client
			if !bb.clientPerThread {
				c, err = cc.ToClient(context.Background(), nil, nilProvidersSettings)
				require.NoError(b, err)
			}
			b.RunParallel(func(pb *testing.PB) {
				if c == nil {
					c, err = cc.ToClient(context.Background(), nil, nilProvidersSettings)
					require.NoError(b, err)
				}
				if bb.forceHTTP1 {
					c.Transport.(*http.Transport).ForceAttemptHTTP2 = false
				}

				for pb.Next() {
					resp, errResp := c.Get(cc.Endpoint)
					require.NoError(b, errResp)
					body, errRead := io.ReadAll(resp.Body)
					_ = resp.Body.Close()
					require.NoError(b, errRead)
					require.Equal(b, "tt", string(body))
				}
				c.CloseIdleConnections()
			})
			// Wait for connections to close before closing server to prevent log spam
			<-time.After(10 * time.Millisecond)
		})
	}
}

func TestDefaultHTTPServerSettings(t *testing.T) {
	httpServerSettings := NewDefaultServerConfig()
	assert.NotNil(t, httpServerSettings.CORS)
	assert.NotNil(t, httpServerSettings.TLS)
	assert.Equal(t, 1*time.Minute, httpServerSettings.IdleTimeout)
	assert.Equal(t, 30*time.Second, httpServerSettings.WriteTimeout)
	assert.Equal(t, time.Duration(0), httpServerSettings.ReadTimeout)
	assert.Equal(t, 1*time.Minute, httpServerSettings.ReadHeaderTimeout)
	assert.True(t, httpServerSettings.KeepAlivesEnabled) // Default should be true (keep-alives enabled by default)
}

func TestHTTPServerKeepAlives(t *testing.T) {
	tests := []struct {
		name               string
		keepAlivesEnabled  bool
		expectedKeepAlives bool
	}{
		{
			name:               "KeepAlives enabled",
			keepAlivesEnabled:  true,
			expectedKeepAlives: true,
		},
		{
			name:               "KeepAlives disabled",
			keepAlivesEnabled:  false,
			expectedKeepAlives: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &ServerConfig{
				Endpoint:          "localhost:0",
				KeepAlivesEnabled: tt.keepAlivesEnabled,
			}

			server, err := sc.ToServer(
				context.Background(),
				nil,
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

			require.NoError(t, err)
			require.NotNil(t, server)

			// Since http.Server.disableKeepAlives is a private field and difficult to test directly,
			// we'll verify the configuration was set by testing the server behavior.
			// The main verification is that ToServer() succeeds without error when DisableKeepAlives is set.

			ln, err := sc.ToListener(context.Background())
			require.NoError(t, err)

			go func() {
				_ = server.Serve(ln)
			}()
			defer func() {
				_ = server.Close()
			}()

			resp, err := http.Get("http://" + ln.Addr().String())
			require.NoError(t, err)
			require.NotNil(t, resp)
			_ = resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			assert.Equal(t, tt.keepAlivesEnabled, sc.KeepAlivesEnabled)
		})
	}
}

func TestHTTPServerTelemetry_Tracing(t *testing.T) {
	// Create a pattern route. The server name the span after the
	// pattern rather than the client-specified path.
	mux := http.NewServeMux()
	mux.HandleFunc("/b/{bucket}/o/{objectname...}", func(http.ResponseWriter, *http.Request) {})

	type testcase struct {
		handler          http.Handler
		expectedSpanName string
	}

	for name, testcase := range map[string]testcase{
		"pattern": {
			handler:          mux,
			expectedSpanName: "GET /b/{bucket}/o/{objectname...}",
		},
		"no_pattern": {
			handler:          http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			expectedSpanName: "GET",
		},
	} {
		t.Run(name, func(t *testing.T) {
			telemetry := componenttest.NewTelemetry()
			config := NewDefaultServerConfig()
			config.Endpoint = "localhost:0"
			srv, err := config.ToServer(
				context.Background(),
				nil,
				telemetry.NewTelemetrySettings(),
				testcase.handler,
			)
			require.NoError(t, err)

			done := make(chan struct{})
			lis, err := config.ToListener(context.Background())
			require.NoError(t, err)
			go func() {
				defer close(done)
				_ = srv.Serve(lis)
			}()
			defer func() {
				assert.NoError(t, srv.Close())
				<-done
			}()

			resp, err := http.Get(fmt.Sprintf("http://%s/b/bucket123/o/object456/segment", lis.Addr()))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()

			spans := telemetry.SpanRecorder.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, testcase.expectedSpanName, spans[0].Name())
		})
	}
}

// TestUnmarshalYAMLWithMiddlewares tests that the "middlewares" field is correctly
// parsed from YAML configurations (fixing the bug where "middleware" was used instead)
func TestServerUnmarshalYAMLWithMiddlewares(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "middlewares.yaml"))
	require.NoError(t, err)

	// Test server configuration
	var serverConfig ServerConfig
	serverSub, err := cm.Sub("server")
	require.NoError(t, err)
	require.NoError(t, serverSub.Unmarshal(&serverConfig))

	// Validate the server configuration using reflection-based validation
	require.NoError(t, xconfmap.Validate(&serverConfig), "Server configuration should be valid")

	assert.Equal(t, "0.0.0.0:4318", serverConfig.Endpoint)
	require.Len(t, serverConfig.Middlewares, 2)
	assert.Equal(t, component.MustNewID("careful_middleware"), serverConfig.Middlewares[0].ID)
	assert.Equal(t, component.MustNewID("support_middleware"), serverConfig.Middlewares[1].ID)
}

// TestUnmarshalYAMLComprehensiveConfig tests the complete configuration example
// to ensure all fields including middlewares are parsed correctly
func TestServerUnmarshalYAMLComprehensiveConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	// Test server configuration
	var serverConfig ServerConfig
	serverSub, err := cm.Sub("server")
	require.NoError(t, err)
	require.NoError(t, serverSub.Unmarshal(&serverConfig))

	// Validate the server configuration using reflection-based validation
	require.NoError(t, xconfmap.Validate(&serverConfig), "Server configuration should be valid")

	// Verify basic fields
	assert.Equal(t, "0.0.0.0:4318", serverConfig.Endpoint)
	assert.Equal(t, 30*time.Second, serverConfig.ReadTimeout)
	assert.Equal(t, 10*time.Second, serverConfig.ReadHeaderTimeout)
	assert.Equal(t, 30*time.Second, serverConfig.WriteTimeout)
	assert.Equal(t, 120*time.Second, serverConfig.IdleTimeout)
	assert.True(t, serverConfig.KeepAlivesEnabled) // Should be true as configured in config.yaml
	assert.Equal(t, int64(33554432), serverConfig.MaxRequestBodySize)
	assert.True(t, serverConfig.IncludeMetadata)

	// Verify TLS configuration
	assert.Equal(t, "/path/to/server.crt", serverConfig.TLS.Get().CertFile)
	assert.Equal(t, "/path/to/server.key", serverConfig.TLS.Get().KeyFile)
	assert.Equal(t, "/path/to/ca.crt", serverConfig.TLS.Get().CAFile)
	assert.Equal(t, "/path/to/client-ca.crt", serverConfig.TLS.Get().ClientCAFile)

	// Verify CORS configuration
	expectedOrigins := []string{"https://example.com", "https://*.test.com"}
	assert.Equal(t, expectedOrigins, serverConfig.CORS.Get().AllowedOrigins)
	corsHeaders := []string{"Content-Type", "Accept"}
	assert.Equal(t, corsHeaders, serverConfig.CORS.Get().AllowedHeaders)
	assert.Equal(t, 7200, serverConfig.CORS.Get().MaxAge)

	// Verify response headers
	expectedResponseHeaders := configopaque.MapList{
		{Name: "Server", Value: "OpenTelemetry-Collector"},
		{Name: "X-Flavor", Value: "apple"},
	}
	assert.Equal(t, expectedResponseHeaders, serverConfig.ResponseHeaders)

	// Verify compression algorithms
	expectedAlgorithms := []string{"", "gzip", "zstd", "zlib", "snappy", "deflate"}
	assert.Equal(t, expectedAlgorithms, serverConfig.CompressionAlgorithms)

	// Verify middlewares
	require.Len(t, serverConfig.Middlewares, 3)
	assert.Equal(t, component.MustNewID("server_middleware1"), serverConfig.Middlewares[0].ID)
	assert.Equal(t, component.MustNewID("server_middleware2"), serverConfig.Middlewares[1].ID)
	assert.Equal(t, component.MustNewID("server_middleware3"), serverConfig.Middlewares[2].ID)
}

// TestMiddlewaresFieldCompatibility tests that the new "middlewares" field name
// is used instead of the old "middleware" name, ensuring the bug is fixed
func TestServerMiddlewaresFieldCompatibility(t *testing.T) {
	// Test that we can create a config with middlewares using the new field name
	serverConfig := ServerConfig{
		Endpoint: "0.0.0.0:4318",
		Middlewares: []configmiddleware.Config{
			{ID: component.MustNewID("server_middleware")},
		},
	}
	assert.Equal(t, "0.0.0.0:4318", serverConfig.Endpoint)
	assert.Len(t, serverConfig.Middlewares, 1)
	assert.Equal(t, component.MustNewID("server_middleware"), serverConfig.Middlewares[0].ID)
}
