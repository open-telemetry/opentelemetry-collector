// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package confighttp

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension/auth"
	"go.opentelemetry.io/collector/extension/auth/authtest"
)

type customRoundTripper struct {
}

var _ http.RoundTripper = (*customRoundTripper)(nil)

func (c *customRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestAllHTTPClientSettings(t *testing.T) {
	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.NewID("testauth"): &authtest.MockClient{ResultRoundTripper: &customRoundTripper{}},
		},
	}

	maxIdleConns := 50
	maxIdleConnsPerHost := 40
	maxConnsPerHost := 45
	idleConnTimeout := 30 * time.Second
	tests := []struct {
		name        string
		settings    HTTPClientSettings
		shouldError bool
	}{
		{
			name: "all_valid_settings",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: false,
				},
				ReadBufferSize:      1024,
				WriteBufferSize:     512,
				MaxIdleConns:        &maxIdleConns,
				MaxIdleConnsPerHost: &maxIdleConnsPerHost,
				MaxConnsPerHost:     &maxConnsPerHost,
				IdleConnTimeout:     &idleConnTimeout,
				CustomRoundTripper:  func(next http.RoundTripper) (http.RoundTripper, error) { return next, nil },
				Compression:         "",
			},
			shouldError: false,
		},
		{
			name: "all_valid_settings_with_none_compression",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: false,
				},
				ReadBufferSize:      1024,
				WriteBufferSize:     512,
				MaxIdleConns:        &maxIdleConns,
				MaxIdleConnsPerHost: &maxIdleConnsPerHost,
				MaxConnsPerHost:     &maxConnsPerHost,
				IdleConnTimeout:     &idleConnTimeout,
				CustomRoundTripper:  func(next http.RoundTripper) (http.RoundTripper, error) { return next, nil },
				Compression:         "none",
			},
			shouldError: false,
		},
		{
			name: "all_valid_settings_with_gzip_compression",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: false,
				},
				ReadBufferSize:      1024,
				WriteBufferSize:     512,
				MaxIdleConns:        &maxIdleConns,
				MaxIdleConnsPerHost: &maxIdleConnsPerHost,
				MaxConnsPerHost:     &maxConnsPerHost,
				IdleConnTimeout:     &idleConnTimeout,
				CustomRoundTripper:  func(next http.RoundTripper) (http.RoundTripper, error) { return next, nil },
				Compression:         "gzip",
			},
			shouldError: false,
		},
		{
			name: "error_round_tripper_returned",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: false,
				},
				ReadBufferSize:     1024,
				WriteBufferSize:    512,
				CustomRoundTripper: func(next http.RoundTripper) (http.RoundTripper, error) { return nil, errors.New("error") },
			},
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tt := componenttest.NewNopTelemetrySettings()
			tt.TracerProvider = nil
			client, err := test.settings.ToClient(host, tt)
			if test.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			switch transport := client.Transport.(type) {
			case *http.Transport:
				assert.EqualValues(t, 1024, transport.ReadBufferSize)
				assert.EqualValues(t, 512, transport.WriteBufferSize)
				assert.EqualValues(t, 50, transport.MaxIdleConns)
				assert.EqualValues(t, 40, transport.MaxIdleConnsPerHost)
				assert.EqualValues(t, 45, transport.MaxConnsPerHost)
				assert.EqualValues(t, 30*time.Second, transport.IdleConnTimeout)
			case *compressRoundTripper:
				assert.EqualValues(t, "gzip", transport.compressionType)
			}
		})
	}
}

func TestPartialHTTPClientSettings(t *testing.T) {
	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.NewID("testauth"): &authtest.MockClient{ResultRoundTripper: &customRoundTripper{}},
		},
	}

	tests := []struct {
		name        string
		settings    HTTPClientSettings
		shouldError bool
	}{
		{
			name: "valid_partial_settings",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				TLSSetting: configtls.TLSClientSetting{
					Insecure: false,
				},
				ReadBufferSize:     1024,
				WriteBufferSize:    512,
				CustomRoundTripper: func(next http.RoundTripper) (http.RoundTripper, error) { return next, nil },
			},
			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tt := componenttest.NewNopTelemetrySettings()
			tt.TracerProvider = nil
			client, err := test.settings.ToClient(host, tt)
			assert.NoError(t, err)
			transport := client.Transport.(*http.Transport)
			assert.EqualValues(t, 1024, transport.ReadBufferSize)
			assert.EqualValues(t, 512, transport.WriteBufferSize)
			assert.EqualValues(t, 100, transport.MaxIdleConns)
			assert.EqualValues(t, 0, transport.MaxIdleConnsPerHost)
			assert.EqualValues(t, 0, transport.MaxConnsPerHost)
			assert.EqualValues(t, 90*time.Second, transport.IdleConnTimeout)

		})
	}
}

func TestDefaultHTTPClientSettings(t *testing.T) {
	httpClientSettings := NewDefaultHTTPClientSettings()
	assert.EqualValues(t, 100, *httpClientSettings.MaxIdleConns)
	assert.EqualValues(t, 90*time.Second, *httpClientSettings.IdleConnTimeout)
}

func TestHTTPClientSettingsError(t *testing.T) {
	host := &mockHost{
		ext: map[component.ID]component.Component{},
	}
	tests := []struct {
		settings HTTPClientSettings
		err      string
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist:",
			settings: HTTPClientSettings{
				Endpoint: "",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
			},
		},
		{
			err: "^failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither",
			settings: HTTPClientSettings{
				Endpoint: "",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "/doesnt/exist",
					},
					Insecure:   false,
					ServerName: "",
				},
			},
		},
		{
			err: "failed to resolve authenticator \"dummy\": authenticator not found",
			settings: HTTPClientSettings{
				Endpoint: "https://localhost:1234/v1/traces",
				Auth:     &configauth.Authentication{AuthenticatorID: component.NewID("dummy")},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToClient(host, componenttest.NewNopTelemetrySettings())
			assert.Regexp(t, test.err, err)
		})
	}
}

func TestHTTPClientSettingWithAuthConfig(t *testing.T) {
	tests := []struct {
		name      string
		shouldErr bool
		settings  HTTPClientSettings
		host      component.Host
	}{
		{
			name: "no_auth_extension_enabled",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     nil,
			},
			shouldErr: false,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					component.NewID("mock"): &authtest.MockClient{
						ResultRoundTripper: &customRoundTripper{},
					},
				},
			},
		},
		{
			name: "with_auth_configuration_and_no_extension",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: component.NewID("dummy")},
			},
			shouldErr: true,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					component.NewID("mock"): &authtest.MockClient{ResultRoundTripper: &customRoundTripper{}},
				},
			},
		},
		{
			name: "with_auth_configuration_and_no_extension_map",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: component.NewID("dummy")},
			},
			shouldErr: true,
			host:      componenttest.NewNopHost(),
		},
		{
			name: "with_auth_configuration_has_extension",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: component.NewID("mock")},
			},
			shouldErr: false,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					component.NewID("mock"): &authtest.MockClient{ResultRoundTripper: &customRoundTripper{}},
				},
			},
		},
		{
			name: "with_auth_configuration_has_err_extension",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorID: component.NewID("mock")},
			},
			shouldErr: true,
			host: &mockHost{
				ext: map[component.ID]component.Component{
					component.NewID("mock"): &authtest.MockClient{
						ResultRoundTripper: &customRoundTripper{}, MustError: true},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := test.settings.ToClient(test.host, componenttest.NewNopTelemetrySettings())
			if test.shouldErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, client)
			if test.settings.Auth != nil {
				_, ok := client.Transport.(*customRoundTripper)
				assert.True(t, ok)
			}
		})
	}
}

func TestHTTPServerSettingsError(t *testing.T) {
	tests := []struct {
		settings HTTPServerSettings
		err      string
	}{
		{
			err: "^failed to load TLS config: failed to load CA CertPool: failed to load CA /doesnt/exist:",
			settings: HTTPServerSettings{
				Endpoint: "localhost:0",
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile: "/doesnt/exist",
					},
				},
			},
		},
		{
			err: "^failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither",
			settings: HTTPServerSettings{
				Endpoint: "localhost:0",
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "/doesnt/exist",
					},
				},
			},
		},
		{
			err: "failed to load client CA CertPool: failed to load CA /doesnt/exist:",
			settings: HTTPServerSettings{
				Endpoint: "localhost:0",
				TLSSetting: &configtls.TLSServerSetting{
					ClientCAFile: "/doesnt/exist",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToListener()
			assert.Regexp(t, test.err, err)
		})
	}
}

func TestHTTPServerWarning(t *testing.T) {
	tests := []struct {
		name     string
		settings HTTPServerSettings
		len      int
	}{
		{
			settings: HTTPServerSettings{
				Endpoint: "0.0.0.0:0",
			},
			len: 1,
		},
		{
			settings: HTTPServerSettings{
				Endpoint: "127.0.0.1:0",
			},
			len: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := componenttest.NewNopTelemetrySettings()
			logger, observed := observer.New(zap.DebugLevel)
			set.Logger = zap.New(logger)

			_, err := test.settings.ToServer(
				componenttest.NewNopHost(),
				set,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, errWrite := fmt.Fprint(w, "test")
					assert.NoError(t, errWrite)
				}))
			require.NoError(t, err)
			require.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), test.len)
		})
	}

}

func TestHttpReception(t *testing.T) {
	tests := []struct {
		name           string
		tlsServerCreds *configtls.TLSServerSetting
		tlsClientCreds *configtls.TLSClientSetting
		hasError       bool
		forceHTTP1     bool
	}{
		{
			name:           "noTLS",
			tlsServerCreds: nil,
			tlsClientCreds: &configtls.TLSClientSetting{
				Insecure: true,
			},
		},
		{
			name: "TLS",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "TLS (HTTP/1.1)",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			forceHTTP1: true,
		},
		{
			name: "NoServerCertificates",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "mTLS",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "client.crt"),
					KeyFile:  filepath.Join("testdata", "client.key"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoClientCertificate",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: filepath.Join("testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "WrongClientCA",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   filepath.Join("testdata", "ca.crt"),
					CertFile: filepath.Join("testdata", "server.crt"),
					KeyFile:  filepath.Join("testdata", "server.key"),
				},
				ClientCAFile: filepath.Join("testdata", "server.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
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
			hss := &HTTPServerSettings{
				Endpoint:   "localhost:0",
				TLSSetting: tt.tlsServerCreds,
			}
			ln, err := hss.ToListener()
			require.NoError(t, err)

			s, err := hss.ToServer(
				componenttest.NewNopHost(),
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, errWrite := fmt.Fprint(w, "test")
					assert.NoError(t, errWrite)
				}))
			require.NoError(t, err)

			go func() {
				_ = s.Serve(ln)
			}()

			// Wait for the servers to start
			<-time.After(10 * time.Millisecond)

			prefix := "https://"
			expectedProto := "HTTP/2.0"
			if tt.tlsClientCreds.Insecure {
				prefix = "http://"
				expectedProto = "HTTP/1.1"
			}

			hcs := &HTTPClientSettings{
				Endpoint:   prefix + ln.Addr().String(),
				TLSSetting: *tt.tlsClientCreds,
			}
			if tt.forceHTTP1 {
				expectedProto = "HTTP/1.1"
				hcs.CustomRoundTripper = func(rt http.RoundTripper) (http.RoundTripper, error) {
					rt.(*http.Transport).ForceAttemptHTTP2 = false
					return rt, nil
				}
			}
			client, errClient := hcs.ToClient(componenttest.NewNopHost(), component.TelemetrySettings{})
			require.NoError(t, errClient)

			resp, errResp := client.Get(hcs.Endpoint)
			if tt.hasError {
				assert.Error(t, errResp)
			} else {
				assert.NoError(t, errResp)
				body, errRead := io.ReadAll(resp.Body)
				assert.NoError(t, errRead)
				assert.Equal(t, "test", string(body))
				assert.Equal(t, expectedProto, resp.Proto)
			}
			require.NoError(t, s.Close())
		})
	}
}

func TestHttpCors(t *testing.T) {
	tests := []struct {
		name string

		CORSSettings

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
			name: "OriginCORS",
			CORSSettings: CORSSettings{
				AllowedOrigins: []string{"allowed-*.com"},
			},
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: false,
		},
		{
			name: "CacheableCORS",
			CORSSettings: CORSSettings{
				AllowedOrigins: []string{"allowed-*.com"},
				MaxAge:         360,
			},
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: false,
		},
		{
			name: "HeaderCORS",
			CORSSettings: CORSSettings{
				AllowedOrigins: []string{"allowed-*.com"},
				AllowedHeaders: []string{"ExtraHeader"},
			},
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hss := &HTTPServerSettings{
				Endpoint: "localhost:0",
				CORS:     &tt.CORSSettings,
			}

			ln, err := hss.ToListener()
			require.NoError(t, err)

			s, err := hss.ToServer(
				componenttest.NewNopHost(),
				componenttest.NewNopTelemetrySettings(),
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			require.NoError(t, err)

			go func() {
				_ = s.Serve(ln)
			}()

			// TODO: make starting server deterministic
			// Wait for the servers to start
			<-time.After(10 * time.Millisecond)

			url := fmt.Sprintf("http://%s", ln.Addr().String())

			expectedStatus := http.StatusNoContent
			if len(tt.AllowedOrigins) == 0 {
				expectedStatus = http.StatusOK
			}
			// Verify allowed domain gets responses that allow CORS.
			verifyCorsResp(t, url, "allowed-origin.com", tt.MaxAge, false, expectedStatus, tt.allowedWorks)

			// Verify allowed domain and extra headers gets responses that allow CORS.
			verifyCorsResp(t, url, "allowed-origin.com", tt.MaxAge, true, expectedStatus, tt.extraHeaderWorks)

			// Verify disallowed domain gets responses that disallow CORS.
			verifyCorsResp(t, url, "disallowed-origin.com", tt.MaxAge, false, expectedStatus, tt.disallowedWorks)

			require.NoError(t, s.Close())
		})
	}
}

func TestHttpCorsInvalidSettings(t *testing.T) {
	hss := &HTTPServerSettings{
		Endpoint: "localhost:0",
		CORS:     &CORSSettings{AllowedHeaders: []string{"some-header"}},
	}

	// This effectively does not enable CORS but should also not cause an error
	s, err := hss.ToServer(
		componenttest.NewNopHost(),
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	require.NoError(t, err)
	require.NotNil(t, s)
	require.NoError(t, s.Close())
}

func TestHttpCorsWithSettings(t *testing.T) {
	hss := &HTTPServerSettings{
		Endpoint: "localhost:0",
		CORS: &CORSSettings{
			AllowedOrigins: []string{"*"},
		},
		Auth: &configauth.Authentication{
			AuthenticatorID: component.NewID("mock"),
		},
	}

	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.NewID("mock"): auth.NewServer(
				auth.WithServerAuthenticate(func(ctx context.Context, headers map[string][]string) (context.Context, error) {
					return ctx, errors.New("Settings failed")
				}),
			),
		},
	}

	srv, err := hss.ToServer(host, componenttest.NewNopTelemetrySettings(), nil)
	require.NoError(t, err)
	require.NotNil(t, srv)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	srv.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func verifyCorsResp(t *testing.T, url string, origin string, maxAge int, extraHeader bool, wantStatus int, wantAllowed bool) {
	req, err := http.NewRequest(http.MethodOptions, url, nil)
	require.NoError(t, err, "Error creating trace OPTIONS request: %v", err)
	req.Header.Set("Origin", origin)
	if extraHeader {
		req.Header.Set("ExtraHeader", "foo")
		req.Header.Set("Access-Control-Request-Headers", "ExtraHeader")
	}
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Error sending OPTIONS to http server: %v", err)

	err = resp.Body.Close()
	if err != nil {
		t.Errorf("Error closing OPTIONS response body, %v", err)
	}

	assert.Equal(t, wantStatus, resp.StatusCode)

	gotAllowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	gotAllowMethods := resp.Header.Get("Access-Control-Allow-Methods")

	wantAllowOrigin := ""
	wantAllowMethods := ""
	wantMaxAge := ""
	if wantAllowed {
		wantAllowOrigin = origin
		wantAllowMethods = "POST"
		if maxAge != 0 {
			wantMaxAge = fmt.Sprintf("%d", maxAge)
		}
	}
	assert.Equal(t, wantAllowOrigin, gotAllowOrigin)
	assert.Equal(t, wantAllowMethods, gotAllowMethods)
	assert.Equal(t, wantMaxAge, resp.Header.Get("Access-Control-Max-Age"))
}

func ExampleHTTPServerSettings() {
	settings := HTTPServerSettings{
		Endpoint: "localhost:443",
	}
	s, err := settings.ToServer(
		componenttest.NewNopHost(),
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err != nil {
		panic(err)
	}

	l, err := settings.ToListener()
	if err != nil {
		panic(err)
	}
	if err = s.Serve(l); err != nil {
		panic(err)
	}
}

func TestHttpHeaders(t *testing.T) {
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
				w.WriteHeader(200)
			}))
			defer server.Close()
			serverURL, _ := url.Parse(server.URL)
			setting := HTTPClientSettings{
				Endpoint:        serverURL.String(),
				TLSSetting:      configtls.TLSClientSetting{},
				ReadBufferSize:  0,
				WriteBufferSize: 0,
				Timeout:         0,
				Headers:         tt.headers,
			}
			client, _ := setting.ToClient(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
			req, err := http.NewRequest(http.MethodGet, setting.Endpoint, nil)
			assert.NoError(t, err)
			_, err = client.Do(req)
			assert.NoError(t, err)
		})
	}
}

func TestContextWithClient(t *testing.T) {
	testCases := []struct {
		desc       string
		input      *http.Request
		doMetadata bool
		expected   client.Info
	}{
		{
			desc:     "request without client IP or headers",
			input:    &http.Request{},
			expected: client.Info{},
		},
		{
			desc: "request with client IP",
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
			desc: "request with client headers, no metadata processing",
			input: &http.Request{
				Header: map[string][]string{"x-test-header": {"test-value"}},
			},
			doMetadata: false,
			expected:   client.Info{},
		},
		{
			desc: "request with client headers",
			input: &http.Request{
				Header: map[string][]string{"x-test-header": {"test-value"}},
			},
			doMetadata: true,
			expected: client.Info{
				Metadata: client.NewMetadata(map[string][]string{"x-test-header": {"test-value"}}),
			},
		},
		{
			desc: "request with Host and client headers",
			input: &http.Request{
				Header: map[string][]string{"x-test-header": {"test-value"}},
				Host:   "localhost:55443",
			},
			doMetadata: true,
			expected: client.Info{
				Metadata: client.NewMetadata(map[string][]string{"x-test-header": {"test-value"}, "Host": {"localhost:55443"}}),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx := contextWithClient(tC.input, tC.doMetadata)
			assert.Equal(t, tC.expected, client.FromContext(ctx))
		})
	}
}

func TestServerAuth(t *testing.T) {
	// prepare
	authCalled := false
	hss := HTTPServerSettings{
		Endpoint: "localhost:0",
		Auth: &configauth.Authentication{
			AuthenticatorID: component.NewID("mock"),
		},
	}

	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.NewID("mock"): auth.NewServer(
				auth.WithServerAuthenticate(func(ctx context.Context, headers map[string][]string) (context.Context, error) {
					authCalled = true
					return ctx, nil
				}),
			),
		},
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	srv, err := hss.ToServer(host, componenttest.NewNopTelemetrySettings(), handler)
	require.NoError(t, err)

	// test
	srv.Handler.ServeHTTP(&httptest.ResponseRecorder{}, httptest.NewRequest("GET", "/", nil))

	// verify
	assert.True(t, handlerCalled)
	assert.True(t, authCalled)
}

func TestInvalidServerAuth(t *testing.T) {
	hss := HTTPServerSettings{
		Auth: &configauth.Authentication{
			AuthenticatorID: component.NewID("non-existing"),
		},
	}

	srv, err := hss.ToServer(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings(), http.NewServeMux())
	require.Error(t, err)
	require.Nil(t, srv)
}

func TestFailedServerAuth(t *testing.T) {
	// prepare
	hss := HTTPServerSettings{
		Endpoint: "localhost:0",
		Auth: &configauth.Authentication{
			AuthenticatorID: component.NewID("mock"),
		},
	}
	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.NewID("mock"): auth.NewServer(
				auth.WithServerAuthenticate(func(ctx context.Context, headers map[string][]string) (context.Context, error) {
					return ctx, errors.New("Settings failed")
				}),
			),
		},
	}

	srv, err := hss.ToServer(host, componenttest.NewNopTelemetrySettings(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	require.NoError(t, err)

	// test
	response := &httptest.ResponseRecorder{}
	srv.Handler.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))

	// verify
	assert.Equal(t, response.Result().StatusCode, http.StatusUnauthorized)
	assert.Equal(t, response.Result().Status, fmt.Sprintf("%v %s", http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)))
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

	tlsServerCreds := &configtls.TLSServerSetting{
		TLSSetting: configtls.TLSSetting{
			CAFile:   filepath.Join("testdata", "ca.crt"),
			CertFile: filepath.Join("testdata", "server.crt"),
			KeyFile:  filepath.Join("testdata", "server.key"),
		},
	}
	tlsClientCreds := &configtls.TLSClientSetting{
		TLSSetting: configtls.TLSSetting{
			CAFile: filepath.Join("testdata", "ca.crt"),
		},
		ServerName: "localhost",
	}

	hss := &HTTPServerSettings{
		Endpoint:   "localhost:0",
		TLSSetting: tlsServerCreds,
	}

	s, err := hss.ToServer(
		componenttest.NewNopHost(),
		componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, errWrite := fmt.Fprint(w, "test")
			require.NoError(b, errWrite)
		}))
	require.NoError(b, err)
	ln, err := hss.ToListener()
	require.NoError(b, err)

	go func() {
		_ = s.Serve(ln)
	}()
	defer func() {
		_ = s.Close()
	}()

	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	for _, bb := range tests {
		hcs := &HTTPClientSettings{
			Endpoint:   "https://" + ln.Addr().String(),
			TLSSetting: *tlsClientCreds,
		}
		if bb.forceHTTP1 {
			hcs.CustomRoundTripper = func(rt http.RoundTripper) (http.RoundTripper, error) {
				rt.(*http.Transport).ForceAttemptHTTP2 = false
				return rt, nil
			}
		}
		b.Run(bb.name, func(b *testing.B) {
			var c *http.Client
			if !bb.clientPerThread {
				c, err = hcs.ToClient(componenttest.NewNopHost(), component.TelemetrySettings{})
				require.NoError(b, err)
			}
			b.RunParallel(func(pb *testing.PB) {
				if c == nil {
					c, err = hcs.ToClient(componenttest.NewNopHost(), component.TelemetrySettings{})
					require.NoError(b, err)
				}
				for pb.Next() {
					resp, errResp := c.Get(hcs.Endpoint)
					require.NoError(b, errResp)
					body, errRead := io.ReadAll(resp.Body)
					_ = resp.Body.Close()
					require.NoError(b, errRead)
					require.Equal(b, "test", string(body))
				}
				c.CloseIdleConnections()
			})
			// Wait for connections to close before closing server to prevent log spam
			<-time.After(10 * time.Millisecond)
		})
	}
}
