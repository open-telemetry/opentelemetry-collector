// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
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
	testAuthID    = component.MustNewID("testauth")
	mockID        = component.MustNewID("mock")
	dummyID       = component.MustNewID("dummy")
	nonExistingID = component.MustNewID("nonexisting")
	// Omit TracerProvider and MeterProvider in TelemetrySettings as otelhttp.Transport cannot be introspected
	nilProvidersSettings = component.TelemetrySettings{Logger: zap.NewNop()}
)

func TestAllHTTPClientSettings(t *testing.T) {
	extensions := map[component.ID]component.Component{
		testAuthID: extensionauthtest.NewNopClient(),
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
				Cookies:              configoptional.Some(CookiesConfig{}),
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
				Cookies:              configoptional.Some(CookiesConfig{}),
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
			client, err := tt.settings.ToClient(context.Background(), extensions, tel)
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
	extensions := map[component.ID]component.Component{
		testAuthID: extensionauthtest.NewNopClient(),
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
			client, err := tt.settings.ToClient(context.Background(), extensions, tel)
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
			client, err := s.ToClient(context.Background(), nil, tel)

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
	extensions := map[component.ID]component.Component{}
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
			_, err := tt.settings.ToClient(context.Background(), extensions, componenttest.NewNopTelemetrySettings())
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
		name       string
		shouldErr  bool
		settings   ClientConfig
		extensions map[component.ID]component.Component
	}{
		{
			name: "no_auth_extension_enabled",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.None[configauth.Config](),
			},
			shouldErr: false,
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopClient(),
			},
		},
		{
			name: "with_auth_configuration_and_no_extension",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: dummyID}),
			},
			shouldErr: true,
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopClient(),
			},
		},
		{
			name: "with_auth_configuration_and_no_extension_map",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: dummyID}),
			},
			shouldErr: true,
		},
		{
			name: "with_auth_configuration_has_extension",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
			},
			shouldErr: false,
			extensions: map[component.ID]component.Component{
				mockID: &mockClient{},
			},
		},
		{
			name: "with_auth_configuration_has_extension_and_headers",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
				Headers: configopaque.MapList{
					{Name: "foo", Value: "bar"},
				},
			},
			shouldErr: false,
			extensions: map[component.ID]component.Component{
				mockID: &mockClient{},
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
			extensions: map[component.ID]component.Component{
				mockID: &mockClient{},
			},
		},
		{
			name: "with_auth_configuration_has_err_extension",
			settings: ClientConfig{
				Endpoint: "localhost:1234",
				Auth:     configoptional.Some(configauth.Config{AuthenticatorID: mockID}),
			},
			shouldErr: true,
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewErr(errors.New("error")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Omit TracerProvider and MeterProvider in TelemetrySettings as otelhttp.Transport cannot be introspected
			client, err := tt.settings.ToClient(context.Background(), tt.extensions, nilProvidersSettings)
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

func TestHttpClientHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers configopaque.MapList
	}{
		{
			name: "with_headers",
			headers: configopaque.MapList{
				{Name: "header1", Value: "value1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers.Iter {
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
			client, _ := setting.ToClient(context.Background(), nil, componenttest.NewNopTelemetrySettings())
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
		headers configopaque.MapList
	}{
		name: "with_host_header",
		headers: configopaque.MapList{
			{Name: "Host", Value: configopaque.String(hostHeader)},
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
		client, _ := setting.ToClient(context.Background(), nil, componenttest.NewNopTelemetrySettings())
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
	client, err := clientConfig.ToClient(context.Background(), nil, settings)
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
	client, err = clientConfig.ToClient(context.Background(), nil, settings)
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

// TestUnmarshalYAMLWithMiddlewares tests that the "middlewares" field is correctly
// parsed from YAML configurations (fixing the bug where "middleware" was used instead)
func TestClientUnmarshalYAMLWithMiddlewares(t *testing.T) {
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
}

// TestUnmarshalYAMLComprehensiveConfig tests the complete configuration example
// to ensure all fields including middlewares are parsed correctly
func TestClientUnmarshalYAMLComprehensiveConfig(t *testing.T) {
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
	expectedHeaders := configopaque.MapList{
		{Name: "User-Agent", Value: "OpenTelemetry-Collector/1.0"},
		{Name: "X-Custom-Header", Value: "custom-value"},
	}
	assert.Equal(t, expectedHeaders, clientConfig.Headers)

	// Verify middlewares
	require.Len(t, clientConfig.Middlewares, 2)
	assert.Equal(t, component.MustNewID("middleware1"), clientConfig.Middlewares[0].ID)
	assert.Equal(t, component.MustNewID("middleware2"), clientConfig.Middlewares[1].ID)
}

// TestMiddlewaresFieldCompatibility tests that the new "middlewares" field name
// is used instead of the old "middleware" name, ensuring the bug is fixed
func TestClientMiddlewaresFieldCompatibility(t *testing.T) {
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
}
