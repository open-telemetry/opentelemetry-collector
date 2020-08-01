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
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/rs/cors"

	"go.opentelemetry.io/collector/config/configtls"
)

type HTTPClientSettings struct {
	// The target URL to send data to (e.g.: http://some.url:9411/v1/trace).
	Endpoint string `mapstructure:"endpoint"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `mapstructure:",squash"`

	// ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
	ReadBufferSize int `mapstructure:"read_buffer_size"`

	// WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
	WriteBufferSize int `mapstructure:"write_buffer_size"`

	// Timeout parameter configures `http.Client.Timeout`.
	Timeout time.Duration `mapstructure:"timeout,omitempty"`
}

func (hcs *HTTPClientSettings) ToClient() (*http.Client, error) {
	tlsCfg, err := hcs.TLSSetting.LoadTLSConfig()
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	if hcs.ReadBufferSize > 0 {
		transport.ReadBufferSize = hcs.ReadBufferSize
	}
	if hcs.WriteBufferSize > 0 {
		transport.WriteBufferSize = hcs.WriteBufferSize
	}
	return &http.Client{
		Transport: transport,
		Timeout:   hcs.Timeout,
	}, nil
}

type HTTPServerSettings struct {
	// Endpoint configures the listening address for the server.
	Endpoint string `mapstructure:"endpoint"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *configtls.TLSServerSetting `mapstructure:"tls_settings, omitempty"`

	// CorsOrigins are the allowed CORS origins for HTTP/JSON requests to grpc-gateway adapter
	// for the OTLP receiver. See github.com/rs/cors
	// An empty list means that CORS is not enabled at all. A wildcard (*) can be
	// used to match any origin or one or more characters of an origin.
	CorsOrigins []string `mapstructure:"cors_allowed_origins"`
}

func (hss *HTTPServerSettings) ToListener() (net.Listener, error) {
	listener, err := net.Listen("tcp", hss.Endpoint)
	if err != nil {
		return nil, err
	}

	if hss.TLSSetting != nil {
		var tlsCfg *tls.Config
		tlsCfg, err = hss.TLSSetting.LoadTLSConfig()
		if err != nil {
			return nil, err
		}
		listener = tls.NewListener(listener, tlsCfg)
	}
	return listener, nil
}

func (hss *HTTPServerSettings) ToServer(handler http.Handler) *http.Server {
	if len(hss.CorsOrigins) > 0 {
		co := cors.Options{AllowedOrigins: hss.CorsOrigins}
		handler = cors.New(co).Handler(handler)
	}
	return &http.Server{
		Handler: handler,
	}
}
