// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"
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

var (
	testAuthID    = component.MustNewID("testauth")
	mockID        = component.MustNewID("mock")
	dummyID       = component.MustNewID("dummy")
	nonExistingID = component.MustNewID("nonexisting")
	// Omit TracerProvider and MeterProvider in TelemetrySettings as otelhttp.Transport cannot be introspected
	nilProvidersSettings = component.TelemetrySettings{Logger: zap.NewNop()}
)

func TestAllHTTPClientSettings(t *testing.T) {
	host := &mockHost{
		ext: map[component.ID]component.Component{
			testAuthID: extensionauthtest.NewNopClient(),
		},
	}

	maxIdleConns := 50
	maxIdleConnsPerHost := 40
	maxConnsPerHost := 45
	idleConnTimeout := 30 * time.Second
	http2PingTimeout := 5 * time.Second
	tests := []struct {
		name        string
		settings    ClientConfig
		shouldError bool
	}{
		{
			name: "all_valid_settings",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				ReadBufferSize:       1024,
				WriteBufferSize:      512,
				MaxIdleConns:         maxIdleConns,
				MaxIdleConnsPerHost:  maxIdleConnsPerHost,
				MaxConnsPerHost:      maxConnsPerHost,
				IdleConnTimeout:      idleConnTimeout,
				Compression:          "",
				DisableKeepAlives:    true,
				Cookies:              CookiesConfig{Enabled: true},
				HTTP2ReadIdleTimeout: idleConnTimeout,
				HTTP2PingTimeout:     http2PingTimeout,
			},
			shouldError: false,
		},
		{
			name: "all_valid_settings_http2_enabled",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				ReadBufferSize:       1024,
				WriteBufferSize:      512,
				MaxIdleConns:         maxIdleConns,
				MaxIdleConnsPerHost:  maxIdleConnsPerHost,
				MaxConnsPerHost:      maxConnsPerHost,
				ForceAttemptHTTP2:    true,
				IdleConnTimeout:      idleConnTimeout,
				Compression:          "",
				DisableKeepAlives:    true,
				Cookies:              CookiesConfig{Enabled: true},
				HTTP2ReadIdleTimeout: idleConnTimeout,
				HTTP2PingTimeout:     http2PingTimeout,
			},
			shouldError: false,
		},
		{
			name: "all_valid_settings_with_none_compression",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				ReadBufferSize:       1024,
				WriteBufferSize:      512,
				MaxIdleConns:         maxIdleConns,
				MaxIdleConnsPerHost:  maxIdleConnsPerHost,
				MaxConnsPerHost:      maxConnsPerHost,
				IdleConnTimeout:      idleConnTimeout,
				Compression:          "none",
				DisableKeepAlives:    true,
				HTTP2ReadIdleTimeout: idleConnTimeout,
				HTTP2PingTimeout:     http2PingTimeout,
			},
			shouldError: false,
		},
		{
			name: "all_valid_settings_with_gzip_compression",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				ReadBufferSize:       1024,
				WriteBufferSize:      512,
				MaxIdleConns:         maxIdleConns,
				MaxIdleConnsPerHost:  maxIdleConnsPerHost,
				MaxConnsPerHost:      maxConnsPerHost,
				IdleConnTimeout:      idleConnTimeout,
				Compression:          "gzip",
				DisableKeepAlives:    true,
				HTTP2ReadIdleTimeout: idleConnTimeout,
				HTTP2PingTimeout:     http2PingTimeout,
			},
			shouldError: false,
		},
		{
			name: "all_valid_settings_http2_health_check",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				ReadBufferSize:       1024,
				WriteBufferSize:      512,
				MaxIdleConns:         maxIdleConns,
				MaxIdleConnsPerHost:  maxIdleConnsPerHost,
				MaxConnsPerHost:      maxConnsPerHost,
				IdleConnTimeout:      idleConnTimeout,
				Compression:          "gzip",
				DisableKeepAlives:    true,
				HTTP2ReadIdleTimeout: idleConnTimeout,
				HTTP2PingTimeout:     http2PingTimeout,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel := componenttest.NewNopTelemetrySettings()
			tel.TracerProvider = nil
			client, err := tt.settings.ToClient(context.Background(), host, tel)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			switch transport := client.Transport.(type) {
			case *http.Transport:
				assert.Equal(t, 1024, transport.ReadBufferSize)
				assert.Equal(t, 512, transport.WriteBufferSize)
				assert.Equal(t, 50, transport.MaxIdleConns)
				assert.Equal(t, 40, transport.MaxIdleConnsPerHost)
				assert.Equal(t, 45, transport.MaxConnsPerHost)
				assert.Equal(t, 30*time.Second, transport.IdleConnTimeout)
				assert.True(t, transport.DisableKeepAlives)
			case *compressRoundTripper:
				assert.EqualValues(t, "gzip", transport.compressionType)
			}
		})
	}
}

func TestPartialHTTPClientSettings(t *testing.T) {
	host := &mockHost{
		ext: map[component.ID]component.Component{
			testAuthID: extensionauthtest.NewNopClient(),
		},
	}

	tests := []struct {
		name        string
		settings    ClientConfig
		shouldError bool
	}{
		{
			name: "valid_partial_settings",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: false,
				},
				ReadBufferSize:  1024,
				WriteBufferSize: 512,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel := componenttest.NewNopTelemetrySettings()
			tel.TracerProvider = nil
			client, err := tt.settings.ToClient(context.Background(), host, tel)
			require.NoError(t, err)
			transport := client.Transport.(*http.Transport)
			assert.Equal(t, 1024, transport.ReadBufferSize)
			assert.Equal(t, 512, transport.WriteBufferSize)
			assert.Equal(t, 0, transport.MaxIdleConns)
			assert.Equal(t, 0, transport.MaxIdleConnsPerHost)
			assert.Equal(t, 0, transport.MaxConnsPerHost)
			assert.EqualValues(t, 0, transport.IdleConnTimeout)
			assert.False(t, transport.DisableKeepAlives)
		})
	}
}

func TestDefaultHTTPClientSettings(t *testing.T) {
	httpClientSettings := NewDefaultClientConfig()
	assert.Equal(t, 100, httpClientSettings.MaxIdleConns)
	assert.Equal(t, 90*time.Second, httpClientSettings.IdleConnTimeout)
}

func TestProxyURL(t *testing.T) {
	testCases := []struct {
		name        string
		proxyURL    string
		expectedURL *url.URL
		err         bool
	}{
		{
			name:        "default config",
			expectedURL: nil,
		},
		{
			name:        "proxy is set",
			proxyURL:    "http://proxy.example.com:8080",
			expectedURL: &url.URL{Scheme: "http", Host: "proxy.example.com:8080"},
		},
		{
			name:     "proxy is invalid",
			proxyURL: "://example.com",
			err:      true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			s := NewDefaultClientConfig()
			s.ProxyURL = tt.proxyURL

			tel := componenttest.NewNopTelemetrySettings()
			tel.TracerProvider = nil
			client, err := s.ToClient(context.Background(), componenttest.NewNopHost(), tel)

			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if err == nil {
				transport := client.Transport.(*http.Transport)
				require.NotNil(t, transport.Proxy)

				url, err := transport.Proxy(&http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}})
				require.NoError(t, err)

				if tt.expectedURL == nil {
					assert.Nil(t, url)
				} else {
					require.NotNil(t, url)
					assert.Equal(t, tt.expectedURL, url)
				}
			}
		})
	}
}

func TestHTTPClientSettingsError(t *testing.T) {
	host := &mockHost{
		ext: map[component.ID]component.Component{},
	}
	tests := []struct {
		settings ClientConfig
		err      string
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool File: failed to load cert /doesnt/exist:",
			settings: ClientConfig{
				Endpoint: "",
				TLS: configtls.ClientConfig{
					Config: configtls.Config{
						CAFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
			},
		},
		{
			err: "^failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither",
			settings: ClientConfig{
				Endpoint: "",
				TLS: configtls.ClientConfig{
					Config: configtls.Config{
						CertFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
			},
		},
		{
			err: "failed to resolve authenticator \"dummy\": authenticator not found",
			settings: ClientConfig{
				Endpoint: "https://localhost:1234/v1/traces",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: dummyID}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.err, func(t *testing.T) {
			_, err := tt.settings.ToClient(context.Background(), host, componenttest.NewNopTelemetrySettings())
			assert.Regexp(t, tt.err, err)
		})
	}
}

var _ http.RoundTripper = &customRoundTripper{}

type customRoundTripper struct{}

func (c *customRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

var (
	_ extensionauth.HTTPClient = (*mockClient)(nil)
	_ extension.Extension      = (*mockClient)(nil)
)

type mockClient struct {
	component.StartFunc
	component.ShutdownFunc
}

// RoundTripper implements extensionauth.HTTPClient.
func (m *mockClient) RoundTripper(http.RoundTripper) (http.RoundTripper, error) {
	return &customRoundTripper{}, nil
}

func TestHTTPClientSettingWithAuthConfig(t *testing.T) {
	tests := []struct {
		name      string
		shouldErr bool
		settings  ClientConfig
		host      component.Host
	}{
		{
			name: "no_auth_extension_enabled",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.None[configauth.Config](),
			},
			shouldErr: false,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					mockID: extensionauthtest.NewNopClient(),
				},
			},
		},
		{
			name: "with_auth_configuration_and_no_extension",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: dummyID}),
			},
			shouldErr: true,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					mockID: extensionauthtest.NewNopClient(),
				},
			},
		},
		{
			name: "with_auth_configuration_and_no_extension_map",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: dummyID}),
			},
			shouldErr: true,
			host:      componenttest.NewNopHost(),
		},
		{
			name: "with_auth_configuration_has_extension",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
			},
			shouldErr: false,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					mockID: &mockClient{},
				},
			},
		},
		{
			name: "with_auth_configuration_has_extension_and_headers",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
				Headers:  map[string]configopaque.String{"foo": "bar"},
			},
			shouldErr: false,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					mockID: &mockClient{},
				},
			},
		},
		{
			name: "with_auth_configuration_has_extension_and_compression",
			settings: ClientConfig{
				Endpoint:    "localhost:1234",
				Auth:        configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
				Compression: configcompression.TypeGzip,
			},
			shouldErr: false,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					mockID: &mockClient{},
				},
			},
		},
		{
			name: "with_auth_configuration_has_err_extension",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
			},
			shouldErr: true,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					mockID: extensionauthtest.NewErr(errors.New("error")),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Omit TracerProvider and MeterProvider in TelemetrySettings as otelhttp.Transport cannot be introspected
			client, err := tt.settings.ToClient(context.Background(), tt.host, nilProvidersSettings)
			if tt.shouldErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, client)
			transport := client.Transport

			// Compression should wrap Auth, unwrap it
			if tt.settings.Compression.IsCompressed() {
				ct, ok := transport.(*compressRoundTripper)
				assert.True(t, ok)
				assert.Equal(t, tt.settings.Compression, ct.compressionType)
				transport = ct.rt
			}

			// Headers should wrap Auth, unwrap it
			if tt.settings.Headers != nil {
				ht, ok := transport.(*headerRoundTripper)
				assert.True(t, ok)
				assert.Equal(t, tt.settings.Headers, ht.headers)
				transport = ht.transport
			}

			if tt.settings.Auth.HasValue() {
				_, ok := transport.(*customRoundTripper)
				assert.True(t, ok)
			}
		})
	}
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
				componenttest.NewNopHost(),
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

			client, errClient := cc.ToClient(context.Background(), componenttest.NewNopHost(), nilProvidersSettings)
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
				componenttest.NewNopHost(),
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
		componenttest.NewNopHost(),
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

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
				return ctx, errors.New("Settings failed")
			}),
		},
	}

	srv, err := sc.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings(), nil)
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
		headers map[string]configopaque.String
	}{
		{
			name:    "noHeaders",
			headers: nil,
		},
		{
			name:    "emptyHeaders",
			headers: map[string]configopaque.String{},
		},
		{
			name: "withHeaders",
			headers: map[string]configopaque.String{
				"x-new-header-1": "value1",
				"x-new-header-2": "value2",
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
				componenttest.NewNopHost(),
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

func verifyHeadersResp(t *testing.T, url string, expected map[string]configopaque.String) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	require.NoError(t, err, "Error creating request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Error sending request to http server")
	require.NotNil(t, resp.Body)
	require.NoError(t, resp.Body.Close(), "Error closing response body")

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	for k, v := range expected {
		assert.Equal(t, string(v), resp.Header.Get(k))
	}
}

func TestHttpClientHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]configopaque.String
	}{
		{
			name: "with_headers",
			headers: map[string]configopaque.String{
				"header1": "value1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					assert.Equal(t, r.Header.Get(k), string(v))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			serverURL, _ := url.Parse(server.URL)
			setting := ClientConfig{
				Endpoint:        serverURL.String(),
				TLS:             configtls.ClientConfig{},
				ReadBufferSize:  0,
				WriteBufferSize: 0,
				Timeout:         0,
				Headers:         tt.headers,
			}
			client, _ := setting.ToClient(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
			req, err := http.NewRequest(http.MethodGet, setting.Endpoint, http.NoBody)
			require.NoError(t, err)
			_, err = client.Do(req)
			assert.NoError(t, err)
		})
	}
}

func TestHttpClientHostHeader(t *testing.T) {
	hostHeader := "th"
	tt := struct {
		name    string
		headers map[string]configopaque.String
	}{
		name: "with_host_header",
		headers: map[string]configopaque.String{
			"Host": configopaque.String(hostHeader),
		},
	}

	t.Run(tt.name, func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, hostHeader, r.Host)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		serverURL, _ := url.Parse(server.URL)
		setting := ClientConfig{
			Endpoint:        serverURL.String(),
			TLS:             configtls.ClientConfig{},
			ReadBufferSize:  0,
			WriteBufferSize: 0,
			Timeout:         0,
			Headers:         tt.headers,
		}
		client, _ := setting.ToClient(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
		req, err := http.NewRequest(http.MethodGet, setting.Endpoint, http.NoBody)
		require.NoError(t, err)
		_, err = client.Do(req)
		assert.NoError(t, err)
	})
}

func TestHttpTransportOptions(t *testing.T) {
	settings := componenttest.NewNopTelemetrySettings()
	// Disable OTel instrumentation so the *http.Transport object is directly accessible
	settings.MeterProvider = nil
	settings.TracerProvider = nil

	clientConfig := NewDefaultClientConfig()
	clientConfig.MaxIdleConns = 100
	clientConfig.IdleConnTimeout = time.Duration(100)
	clientConfig.MaxConnsPerHost = 100
	clientConfig.MaxIdleConnsPerHost = 100
	client, err := clientConfig.ToClient(context.Background(), &mockHost{}, settings)
	require.NoError(t, err)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "client.Transport is not an *http.Transport")
	require.Equal(t, 100, transport.MaxIdleConns)
	require.Equal(t, time.Duration(100), transport.IdleConnTimeout)
	require.Equal(t, 100, transport.MaxConnsPerHost)
	require.Equal(t, 100, transport.MaxIdleConnsPerHost)

	clientConfig = NewDefaultClientConfig()
	clientConfig.MaxIdleConns = 0
	clientConfig.IdleConnTimeout = 0
	clientConfig.MaxConnsPerHost = 0
	clientConfig.IdleConnTimeout = time.Duration(0)
	client, err = clientConfig.ToClient(context.Background(), &mockHost{}, settings)
	require.NoError(t, err)
	transport, ok = client.Transport.(*http.Transport)
	require.True(t, ok, "client.Transport is not an *http.Transport")
	require.Equal(t, 0, transport.MaxIdleConns)
	require.Equal(t, time.Duration(0), transport.IdleConnTimeout)
	require.Equal(t, 0, transport.MaxConnsPerHost)
	require.Equal(t, 0, transport.MaxIdleConnsPerHost)
}

func TestContextWithClient(t *testing.T) {
	testCases := []struct {
		name       string
		input      *http.Request
		doMetadata bool
		expected   client.Info
	}{
		{
			name:     "request without client IP or headers",
			input:    &http.Request{},
			expected: client.Info{},
		},
		{
			name: "request with client IP",
			input: &http.Request{
				RemoteAddr: "1.2.3.4:55443",
			},
			expected: client.Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			},
		},
		{
			name: "request with client headers, no metadata processing",
			input: &http.Request{
				Header: map[string][]string{"x-tt-header": {"tt-value"}},
			},
			doMetadata: false,
			expected:   client.Info{},
		},
		{
			name: "request with client headers",
			input: &http.Request{
				Header: map[string][]string{"x-tt-header": {"tt-value"}},
			},
			doMetadata: true,
			expected: client.Info{
				Metadata: client.NewMetadata(map[string][]string{"x-tt-header": {"tt-value"}}),
			},
		},
		{
			name: "request with Host and client headers",
			input: &http.Request{
				Header: map[string][]string{"x-tt-header": {"tt-value"}},
				Host:   "localhost:55443",
			},
			doMetadata: true,
			expected: client.Info{
				Metadata: client.NewMetadata(map[string][]string{"x-tt-header": {"tt-value"}, "Host": {"localhost:55443"}}),
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := contextWithClient(tt.input, tt.doMetadata)
			assert.Equal(t, tt.expected, client.FromContext(ctx))
		})
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

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
				authCalled = true
				return ctx, nil
			}),
		},
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	})

	srv, err := sc.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings(), handler)
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

	srv, err := sc.ToServer(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings(), http.NewServeMux())
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
	host := &mockHost{
		ext: map[component.ID]component.Component{
			mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
				return ctx, errors.New("invalid authorization")
			}),
		},
	}

	srv, err := sc.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
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
	host := &mockHost{
		ext: map[component.ID]component.Component{
			mockID: newMockAuthServer(func(ctx context.Context, _ map[string][]string) (context.Context, error) {
				return ctx, errors.New("invalid authorization")
			}),
		},
	}

	eh := func(w http.ResponseWriter, _ *http.Request, err string, statusCode int) {
		assert.Equal(t, http.StatusUnauthorized, statusCode)
		// custom error handler uses real error string
		assert.Equal(t, "invalid authorization", err)
		// custom error handler changes returned status code
		http.Error(w, err, http.StatusInternalServerError)
	}

	srv, err := sc.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), WithErrorHandler(eh))
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
		componenttest.NewNopHost(),
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
		componenttest.NewNopHost(),
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
		componenttest.NewNopHost(),
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
				componenttest.NewNopHost(),
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

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mockID: newMockAuthServer(func(ctx context.Context, sources map[string][]string) (context.Context, error) {
				require.Len(t, sources, 1)
				assert.Equal(t, "1", sources["auth"][0])
				authCalled = true
				return ctx, nil
			}),
		},
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	})

	srv, err := sc.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings(), handler)
	require.NoError(t, err)

	// tt
	srv.Handler.ServeHTTP(&httptest.ResponseRecorder{}, httptest.NewRequest(http.MethodGet, "/?auth=1", http.NoBody))

	// verify
	assert.True(t, handlerCalled)
	assert.True(t, authCalled)
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
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
		componenttest.NewNopHost(),
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
				c, err = cc.ToClient(context.Background(), componenttest.NewNopHost(), nilProvidersSettings)
				require.NoError(b, err)
			}
			b.RunParallel(func(pb *testing.PB) {
				if c == nil {
					c, err = cc.ToClient(context.Background(), componenttest.NewNopHost(), nilProvidersSettings)
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
	assert.NotNil(t, httpServerSettings.ResponseHeaders)
	assert.NotNil(t, httpServerSettings.CORS)
	assert.NotNil(t, httpServerSettings.TLS)
	assert.Equal(t, 1*time.Minute, httpServerSettings.IdleTimeout)
	assert.Equal(t, 30*time.Second, httpServerSettings.WriteTimeout)
	assert.Equal(t, time.Duration(0), httpServerSettings.ReadTimeout)
	assert.Equal(t, 1*time.Minute, httpServerSettings.ReadHeaderTimeout)
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
				componenttest.NewNopHost(),
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
func TestUnmarshalYAMLWithMiddlewares(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "middlewares.yaml"))
	require.NoError(t, err)

	// Test client configuration
	var clientConfig ClientConfig
	clientSub, err := cm.Sub("client")
	require.NoError(t, err)
	require.NoError(t, clientSub.Unmarshal(&clientConfig))

	// Validate the client configuration using reflection-based validation
	require.NoError(t, xconfmap.Validate(&clientConfig), "Client configuration should be valid")

	assert.Equal(t, "http://localhost:4318/v1/traces", clientConfig.Endpoint)
	require.Len(t, clientConfig.Middlewares, 2)
	assert.Equal(t, component.MustNewID("fancy_middleware"), clientConfig.Middlewares[0].ID)
	assert.Equal(t, component.MustNewID("careful_middleware"), clientConfig.Middlewares[1].ID)

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
func TestUnmarshalYAMLComprehensiveConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	// Test client configuration
	var clientConfig ClientConfig
	clientSub, err := cm.Sub("client")
	require.NoError(t, err)
	require.NoError(t, clientSub.Unmarshal(&clientConfig))

	// Validate the client configuration using reflection-based validation
	require.NoError(t, xconfmap.Validate(&clientConfig), "Client configuration should be valid")

	// Verify basic fields
	assert.Equal(t, "http://example.com:4318/v1/traces", clientConfig.Endpoint)
	assert.Equal(t, "http://proxy.example.com:8080", clientConfig.ProxyURL)
	assert.Equal(t, 30*time.Second, clientConfig.Timeout)
	assert.Equal(t, 4096, clientConfig.ReadBufferSize)
	assert.Equal(t, 4096, clientConfig.WriteBufferSize)
	assert.Equal(t, configcompression.TypeGzip, clientConfig.Compression)

	// Verify TLS configuration
	assert.False(t, clientConfig.TLS.Insecure)
	assert.Equal(t, "/path/to/client.crt", clientConfig.TLS.CertFile)
	assert.Equal(t, "/path/to/client.key", clientConfig.TLS.KeyFile)
	assert.Equal(t, "/path/to/ca.crt", clientConfig.TLS.CAFile)
	assert.Equal(t, "example.com", clientConfig.TLS.ServerName)

	// Verify headers
	expectedHeaders := map[string]configopaque.String{
		"User-Agent":      "OpenTelemetry-Collector/1.0",
		"X-Custom-Header": "custom-value",
	}
	assert.Equal(t, expectedHeaders, clientConfig.Headers)

	// Verify middlewares
	require.Len(t, clientConfig.Middlewares, 2)
	assert.Equal(t, component.MustNewID("middleware1"), clientConfig.Middlewares[0].ID)
	assert.Equal(t, component.MustNewID("middleware2"), clientConfig.Middlewares[1].ID)

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
	expectedResponseHeaders := map[string]configopaque.String{
		"Server":   "OpenTelemetry-Collector",
		"X-Flavor": "apple",
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
func TestMiddlewaresFieldCompatibility(t *testing.T) {
	// Test that we can create a config with middlewares using the new field name
	clientConfig := ClientConfig{
		Endpoint: "http://localhost:4318",
		Middlewares: []configmiddleware.Config{
			{ID: component.MustNewID("test_middleware")},
		},
	}
	assert.Equal(t, "http://localhost:4318", clientConfig.Endpoint)
	assert.Len(t, clientConfig.Middlewares, 1)
	assert.Equal(t, component.MustNewID("test_middleware"), clientConfig.Middlewares[0].ID)

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
