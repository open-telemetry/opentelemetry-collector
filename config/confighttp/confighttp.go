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

// Package confighttp defines the HTTP configuration settings.
package confighttp

import (
	"net/http"

	"github.com/rs/cors"

	"go.opentelemetry.io/collector/config/configtls"
)

// HTTPServerSettings defines common settings for a HTTP server configuration.
type HTTPServerSettings struct {

	// Addr the address for the server to listen on.
	Addr string `mapstructure:"addr"`

	// TLSCredentials Configures the HTTP server to use TLS.
	// The default value is nil, which will cause the HTTP server to not use TLS.
	TLSCredentials *configtls.TLSSetting `mapstructure:"tls_credentials, omitempty"`

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	MaxHeaderBytes int `mapstructure:"max_header_bytes"`

	// CorsOrigins are the allowed CORS origins for HTTP requests.
	// An empty list means that CORS is not enabled at all. A wildcard (*) can be
	// used to match any origin or one or more characters of an origin.
	CorsOrigins []string `mapstructure:"cors_allowed_origins"`
}

func HTTPServerSettingsToHTTPServer(settings HTTPServerSettings, h http.Handler) *http.Server {
	if h == nil {
		h = http.DefaultServeMux
	}
	s := &http.Server{
		Addr:           settings.Addr,
		MaxHeaderBytes: settings.MaxHeaderBytes,
	}
	if len(settings.CorsOrigins) > 0 {
		co := cors.Options{AllowedOrigins: settings.CorsOrigins}
		h = cors.New(co).Handler(h)
	}
	s.Handler = h
	return s
}
