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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configtls"
)

type customRoundTripper struct {
}

var _ http.RoundTripper = (*customRoundTripper)(nil)

func (c *customRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestAllHTTPClientSettings(t *testing.T) {
	ext := map[config.ComponentID]component.Extension{
		config.NewID("testauth"): &configauth.MockClientAuthenticator{ResultRoundTripper: &customRoundTripper{}},
	}
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
				ReadBufferSize:     1024,
				WriteBufferSize:    512,
				CustomRoundTripper: func(next http.RoundTripper) (http.RoundTripper, error) { return next, nil },
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
			client, err := test.settings.ToClient(ext)
			if test.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			transport := client.Transport.(*http.Transport)
			assert.EqualValues(t, 1024, transport.ReadBufferSize)
			assert.EqualValues(t, 512, transport.WriteBufferSize)
		})
	}
}

func TestHTTPClientSettingsError(t *testing.T) {
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
				Auth:     &configauth.Authentication{AuthenticatorName: "dummy"},
			},
		},
		{
			err: "idStr must have non empty type",
			settings: HTTPClientSettings{
				Endpoint: "https://localhost:1234/v1/traces",
				Auth:     &configauth.Authentication{AuthenticatorName: ""},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToClient(map[config.ComponentID]component.Extension{})
			assert.Regexp(t, test.err, err)
		})
	}
}

func TestHTTPClientSettingWithAuthConfig(t *testing.T) {
	tests := []struct {
		name         string
		shouldErr    bool
		settings     HTTPClientSettings
		extensionMap map[config.ComponentID]component.Extension
	}{
		{
			name: "no_auth_extension_enabled",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     nil,
			},
			shouldErr: false,
			extensionMap: map[config.ComponentID]component.Extension{
				config.NewID("mock"): &configauth.MockClientAuthenticator{
					ResultRoundTripper: &customRoundTripper{},
				},
			},
		},
		{
			name: "with_auth_configuration_and_no_extension",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorName: "dummy"},
			},
			shouldErr: true,
			extensionMap: map[config.ComponentID]component.Extension{
				config.NewID("mock"): &configauth.MockClientAuthenticator{ResultRoundTripper: &customRoundTripper{}},
			},
		},
		{
			name: "with_auth_configuration_and_no_extension_map",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorName: "dummy"},
			},
			shouldErr: true,
		},
		{
			name: "with_auth_configuration_has_extension",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorName: "mock"},
			},
			shouldErr: false,
			extensionMap: map[config.ComponentID]component.Extension{
				config.NewID("mock"): &configauth.MockClientAuthenticator{ResultRoundTripper: &customRoundTripper{}},
			},
		},
		{
			name: "with_auth_configuration_has_err_extension",
			settings: HTTPClientSettings{
				Endpoint: "localhost:1234",
				Auth:     &configauth.Authentication{AuthenticatorName: "mock"},
			},
			shouldErr: true,
			extensionMap: map[config.ComponentID]component.Extension{
				config.NewID("mock"): &configauth.MockClientAuthenticator{
					ResultRoundTripper: &customRoundTripper{}, MustError: true},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := test.settings.ToClient(test.extensionMap)
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
				Endpoint: "",
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
				Endpoint: "",
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "/doesnt/exist",
					},
				},
			},
		},
		{
			err: "^failed to load TLS config: failed to load client CA CertPool: failed to load CA /doesnt/exist:",
			settings: HTTPServerSettings{
				Endpoint: "",
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

func TestHttpReception(t *testing.T) {
	tests := []struct {
		name           string
		tlsServerCreds *configtls.TLSServerSetting
		tlsClientCreds *configtls.TLSClientSetting
		hasError       bool
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
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoServerCertificates",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "mTLS",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
				ClientCAFile: path.Join(".", "testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "client.crt"),
					KeyFile:  path.Join(".", "testdata", "client.key"),
				},
				ServerName: "localhost",
			},
		},
		{
			name: "NoClientCertificate",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
				ClientCAFile: path.Join(".", "testdata", "ca.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: path.Join(".", "testdata", "ca.crt"),
				},
				ServerName: "localhost",
			},
			hasError: true,
		},
		{
			name: "WrongClientCA",
			tlsServerCreds: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "server.crt"),
					KeyFile:  path.Join(".", "testdata", "server.key"),
				},
				ClientCAFile: path.Join(".", "testdata", "server.crt"),
			},
			tlsClientCreds: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile:   path.Join(".", "testdata", "ca.crt"),
					CertFile: path.Join(".", "testdata", "client.crt"),
					KeyFile:  path.Join(".", "testdata", "client.key"),
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
			assert.NoError(t, err)
			s := hss.ToServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, errWrite := fmt.Fprint(w, "test")
				assert.NoError(t, errWrite)
			}))

			go func() {
				_ = s.Serve(ln)
			}()

			// Wait for the servers to start
			<-time.After(10 * time.Millisecond)

			prefix := "https://"
			if tt.tlsClientCreds.Insecure {
				prefix = "http://"
			}

			hcs := &HTTPClientSettings{
				Endpoint:   prefix + ln.Addr().String(),
				TLSSetting: *tt.tlsClientCreds,
			}
			client, errClient := hcs.ToClient(map[config.ComponentID]component.Extension{})
			assert.NoError(t, errClient)
			resp, errResp := client.Get(hcs.Endpoint)
			if tt.hasError {
				assert.Error(t, errResp)
			} else {
				assert.NoError(t, errResp)
				body, errRead := ioutil.ReadAll(resp.Body)
				assert.NoError(t, errRead)
				assert.Equal(t, "test", string(body))
			}
			require.NoError(t, s.Close())
		})
	}
}

func TestHttpCors(t *testing.T) {
	tests := []struct {
		name             string
		CorsOrigins      []string
		CorsHeaders      []string
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
			name:             "OriginCORS",
			CorsOrigins:      []string{"allowed-*.com"},
			CorsHeaders:      []string{},
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: false,
		},
		{
			name:             "HeaderCORS",
			CorsOrigins:      []string{"allowed-*.com"},
			CorsHeaders:      []string{"ExtraHeader"},
			allowedWorks:     true,
			disallowedWorks:  false,
			extraHeaderWorks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hss := &HTTPServerSettings{
				Endpoint:    "localhost:0",
				CorsOrigins: tt.CorsOrigins,
				CorsHeaders: tt.CorsHeaders,
			}

			ln, err := hss.ToListener()
			assert.NoError(t, err)
			s := hss.ToServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			go func() {
				_ = s.Serve(ln)
			}()

			// TODO: make starting server deterministic
			// Wait for the servers to start
			<-time.After(10 * time.Millisecond)

			url := fmt.Sprintf("http://%s", ln.Addr().String())

			expectedStatus := http.StatusNoContent
			if len(tt.CorsOrigins) == 0 {
				expectedStatus = http.StatusOK
			}
			// Verify allowed domain gets responses that allow CORS.
			verifyCorsResp(t, url, "allowed-origin.com", false, expectedStatus, tt.allowedWorks)

			// Verify allowed domain and extra headers gets responses that allow CORS.
			verifyCorsResp(t, url, "allowed-origin.com", true, expectedStatus, tt.extraHeaderWorks)

			// Verify disallowed domain gets responses that disallow CORS.
			verifyCorsResp(t, url, "disallowed-origin.com", false, expectedStatus, tt.disallowedWorks)

			require.NoError(t, s.Close())
		})
	}
}

func TestHttpCorsInvalidSettings(t *testing.T) {
	hss := &HTTPServerSettings{
		Endpoint:    "localhost:0",
		CorsHeaders: []string{"some-header"},
	}

	// This effectively does not enable CORS but should also not cause an error
	s := hss.ToServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	require.NotNil(t, s)
	require.NoError(t, s.Close())
}

func verifyCorsResp(t *testing.T, url string, origin string, extraHeader bool, wantStatus int, wantAllowed bool) {
	req, err := http.NewRequest("OPTIONS", url, nil)
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
	if wantAllowed {
		wantAllowOrigin = origin
		wantAllowMethods = "POST"
	}
	assert.Equal(t, wantAllowOrigin, gotAllowOrigin)
	assert.Equal(t, wantAllowMethods, gotAllowMethods)
}

func ExampleHTTPServerSettings() {
	settings := HTTPServerSettings{
		Endpoint: ":443",
	}
	s := settings.ToServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
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
		headers map[string]string
	}{
		{
			"with_headers",
			map[string]string{
				"header1": "value1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					assert.Equal(t, r.Header.Get(k), v)
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
				Headers: map[string]string{
					"header1": "value1",
				},
			}
			client, _ := setting.ToClient(map[config.ComponentID]component.Extension{})
			req, err := http.NewRequest("GET", setting.Endpoint, nil)
			assert.NoError(t, err)
			_, err = client.Do(req)
			assert.NoError(t, err)
		})
	}
}
