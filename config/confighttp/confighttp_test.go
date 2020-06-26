// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package confighttp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtls"
)

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
	}
	for _, test := range tests {
		t.Run(test.err, func(t *testing.T) {
			_, err := test.settings.ToClient()
			assert.Regexp(t, test.err, err)
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

			prefix := "https://"
			if tt.tlsClientCreds.Insecure {
				prefix = "http://"
			}

			hcs := &HTTPClientSettings{
				Endpoint:   prefix + ln.Addr().String(),
				TLSSetting: *tt.tlsClientCreds,
			}
			client, errClient := hcs.ToClient()
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
