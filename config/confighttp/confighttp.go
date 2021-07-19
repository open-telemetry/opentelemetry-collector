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
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/internal/middleware"
)

// HTTPClientSettings defines settings for creating an HTTP client.
type HTTPClientSettings struct {
	// The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
	Endpoint string `mapstructure:"endpoint"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `mapstructure:",squash"`

	// ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
	ReadBufferSize int `mapstructure:"read_buffer_size"`

	// WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
	WriteBufferSize int `mapstructure:"write_buffer_size"`

	// Timeout parameter configures `http.Client.Timeout`.
	Timeout time.Duration `mapstructure:"timeout,omitempty"`

	// Additional headers attached to each HTTP request sent by the client.
	// Existing header values are overwritten if collision happens.
	Headers map[string]string `mapstructure:"headers,omitempty"`

	// Custom Round Tripper to allow for individual components to intercept HTTP requests
	CustomRoundTripper func(next http.RoundTripper) (http.RoundTripper, error)

	// Auth configuration for outgoing HTTP calls.
	Auth *configauth.Authentication `mapstructure:"auth,omitempty"`
}

// ToClient creates an HTTP client.
func (hcs *HTTPClientSettings) ToClient(ext map[config.ComponentID]component.Extension) (*http.Client, error) {
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

	clientTransport := (http.RoundTripper)(transport)
	if len(hcs.Headers) > 0 {
		clientTransport = &headerRoundTripper{
			transport: transport,
			headers:   hcs.Headers,
		}
	}

	if hcs.Auth != nil {
		if ext == nil {
			return nil, fmt.Errorf("extensions configuration not found")
		}

		componentID, cperr := config.NewIDFromString(hcs.Auth.AuthenticatorName)
		if cperr != nil {
			return nil, cperr
		}

		httpCustomAuthRoundTripper, aerr := configauth.GetHTTPClientAuthenticator(ext, componentID)
		if aerr != nil {
			return nil, aerr
		}

		clientTransport, err = httpCustomAuthRoundTripper.RoundTripper(clientTransport)
		if err != nil {
			return nil, err
		}
	}

	if hcs.CustomRoundTripper != nil {
		clientTransport, err = hcs.CustomRoundTripper(clientTransport)
		if err != nil {
			return nil, err
		}
	}

	return &http.Client{
		Transport: clientTransport,
		Timeout:   hcs.Timeout,
	}, nil
}

// Custom RoundTripper that adds headers.
type headerRoundTripper struct {
	transport http.RoundTripper
	headers   map[string]string
}

// RoundTrip is a custom RoundTripper that adds headers to the request.
func (interceptor *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range interceptor.headers {
		req.Header.Set(k, v)
	}
	// Send the request to next transport.
	return interceptor.transport.RoundTrip(req)
}

// HTTPServerSettings defines settings for creating an HTTP server.
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

	// CorsHeaders are the allowed CORS headers for HTTP/JSON requests to grpc-gateway adapter
	// for the OTLP receiver. See github.com/rs/cors
	// CORS needs to be enabled first by providing a non-empty list in CorsOrigins
	// A wildcard (*) can be used to match any header.
	CorsHeaders []string `mapstructure:"cors_allowed_headers"`
}

// ToListener creates a net.Listener.
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

// toServerOptions has options that change the behavior of the HTTP server
// returned by HTTPServerSettings.ToServer().
type toServerOptions struct {
	errorHandler middleware.ErrorHandler
}

// ToServerOption is an option to change the behavior of the HTTP server
// returned by HTTPServerSettings.ToServer().
type ToServerOption func(opts *toServerOptions)

// WithErrorHandler overrides the HTTP error handler that gets invoked
// when there is a failure inside middleware.HTTPContentDecompressor.
func WithErrorHandler(e middleware.ErrorHandler) ToServerOption {
	return func(opts *toServerOptions) {
		opts.errorHandler = e
	}
}

// ToServer creates an http.Server from settings object.
func (hss *HTTPServerSettings) ToServer(handler http.Handler, opts ...ToServerOption) *http.Server {
	serverOpts := &toServerOptions{}
	for _, o := range opts {
		o(serverOpts)
	}

	handler = middleware.HTTPContentDecompressor(
		handler,
		middleware.WithErrorHandler(serverOpts.errorHandler),
	)

	if len(hss.CorsOrigins) > 0 {
		co := cors.Options{
			AllowedOrigins:   hss.CorsOrigins,
			AllowCredentials: true,
			AllowedHeaders:   hss.CorsHeaders,
		}
		handler = cors.New(co).Handler(handler)
	}
	// TODO: emit a warning when non-empty CorsHeaders and empty CorsOrigins.

	// Enable OpenTelemetry observability plugin.
	// TODO: Consider to use component ID string as prefix for all the operations.
	handler = otelhttp.NewHandler(
		handler,
		"",
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.URL.Path
		}),
	)

	return &http.Server{
		Handler: handler,
	}
}
