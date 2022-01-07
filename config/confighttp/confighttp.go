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

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configtls"
)

type CompressionType string

const (
	headerContentEncoding                 = "Content-Encoding"
	CompressionGzip       CompressionType = "gzip"
	CompressionZlib       CompressionType = "zlib"
	CompressionDeflate    CompressionType = "deflate"
	CompressionSnappy     CompressionType = "snappy"
	CompressionZstd       CompressionType = "zstd"
	CompressionNone       CompressionType = "none"
	CompressionEmpty      CompressionType = ""
)

// HTTPClientSettings defines settings for creating an HTTP client.
type HTTPClientSettings struct {
	// The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
	Endpoint string `mapstructure:"endpoint"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `mapstructure:"tls,omitempty"`

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

	// The compression key for supported compression types within collector.
	Compression CompressionType `mapstructure:"compression"`

	// MaxIdleConns is used to set a limit to the maximum idle HTTP connections the client can keep open.
	// There's an already set value, and we want to override it only if an explicit value provided
	MaxIdleConns *int `mapstructure:"max_idle_conns"`

	// MaxIdleConnsPerHost is used to set a limit to the maximum idle HTTP connections the host can keep open.
	// There's an already set value, and we want to override it only if an explicit value provided
	MaxIdleConnsPerHost *int `mapstructure:"max_idle_conns_per_host"`

	// MaxConnsPerHost limits the total number of connections per host, including connections in the dialing,
	// active, and idle states.
	// There's an already set value, and we want to override it only if an explicit value provided
	MaxConnsPerHost *int `mapstructure:"max_conns_per_host"`

	// IdleConnTimeout is the maximum amount of time a connection will remain open before closing itself.
	// There's an already set value, and we want to override it only if an explicit value provided
	IdleConnTimeout *time.Duration `mapstructure:"idle_conn_timeout"`
}

// DefaultHTTPClientSettings returns HTTPClientSettings type object with
// the default values of 'MaxIdleConns' and 'IdleConnTimeout'.
// Other config options are not added as they are initialized with 'zero value' by GoLang as default.
// We encourage to use this function to create an object of HTTPClientSettings.
func DefaultHTTPClientSettings() HTTPClientSettings {
	// The default values are taken from the values of 'DefaultTransport' of 'http' package.
	maxIdleConns := 100
	idleConnTimeout := 90 * time.Second

	return HTTPClientSettings{
		MaxIdleConns:    &maxIdleConns,
		IdleConnTimeout: &idleConnTimeout,
	}
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

	if hcs.MaxIdleConns != nil {
		transport.MaxIdleConns = *hcs.MaxIdleConns
	}

	if hcs.MaxIdleConnsPerHost != nil {
		transport.MaxIdleConnsPerHost = *hcs.MaxIdleConnsPerHost
	}

	if hcs.MaxConnsPerHost != nil {
		transport.MaxConnsPerHost = *hcs.MaxConnsPerHost
	}

	if hcs.IdleConnTimeout != nil {
		transport.IdleConnTimeout = *hcs.IdleConnTimeout
	}

	clientTransport := (http.RoundTripper)(transport)
	if len(hcs.Headers) > 0 {
		clientTransport = &headerRoundTripper{
			transport: transport,
			headers:   hcs.Headers,
		}
	}

	// Compress the body using specified compression methods if non-empty string is provided.
	// Supporting gzip, zlib, deflate, snappy, and zstd; none is treated as uncompressed.
	if hcs.Compression != CompressionEmpty && hcs.Compression != CompressionNone {
		clientTransport = newCompressRoundTripper(clientTransport, hcs.Compression)
	}

	if hcs.Auth != nil {
		if ext == nil {
			return nil, fmt.Errorf("extensions configuration not found")
		}

		httpCustomAuthRoundTripper, aerr := hcs.Auth.GetClientAuthenticator(ext)
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
	TLSSetting *configtls.TLSServerSetting `mapstructure:"tls, omitempty"`

	// CORS configures the server for HTTP cross-origin resource sharing (CORS).
	CORS *CORSSettings `mapstructure:"cors,omitempty"`

	// Auth for this receiver
	Auth *configauth.Authentication `mapstructure:"auth,omitempty"`
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
	errorHandler
}

// ToServerOption is an option to change the behavior of the HTTP server
// returned by HTTPServerSettings.ToServer().
type ToServerOption func(opts *toServerOptions)

// WithErrorHandler overrides the HTTP error handler that gets invoked
// when there is a failure inside httpContentDecompressor.
func WithErrorHandler(e errorHandler) ToServerOption {
	return func(opts *toServerOptions) {
		opts.errorHandler = e
	}
}

// ToServer creates an http.Server from settings object.
func (hss *HTTPServerSettings) ToServer(host component.Host, settings component.TelemetrySettings, handler http.Handler, opts ...ToServerOption) (*http.Server, error) {
	serverOpts := &toServerOptions{}
	for _, o := range opts {
		o(serverOpts)
	}

	handler = httpContentDecompressor(
		handler,
		withErrorHandlerForDecompressor(serverOpts.errorHandler),
	)

	if hss.CORS != nil && len(hss.CORS.AllowedOrigins) > 0 {
		co := cors.Options{
			AllowedOrigins:   hss.CORS.AllowedOrigins,
			AllowCredentials: true,
			AllowedHeaders:   hss.CORS.AllowedHeaders,
			MaxAge:           hss.CORS.MaxAge,
		}
		handler = cors.New(co).Handler(handler)
	}
	// TODO: emit a warning when non-empty CorsHeaders and empty CorsOrigins.

	if hss.Auth != nil {
		authenticator, err := hss.Auth.GetServerAuthenticator(host.GetExtensions())
		if err != nil {
			return nil, err
		}

		handler = authInterceptor(handler, authenticator.Authenticate)
	}

	// Enable OpenTelemetry observability plugin.
	// TODO: Consider to use component ID string as prefix for all the operations.
	handler = otelhttp.NewHandler(
		handler,
		"",
		otelhttp.WithTracerProvider(settings.TracerProvider),
		otelhttp.WithMeterProvider(settings.MeterProvider),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.URL.Path
		}),
	)

	// wrap the current handler in an interceptor that will add client.Info to the request's context
	handler = &clientInfoHandler{
		next: handler,
	}

	return &http.Server{
		Handler: handler,
	}, nil
}

// CORSSettings configures a receiver for HTTP cross-origin resource sharing (CORS).
// See the underlying https://github.com/rs/cors package for details.
type CORSSettings struct {
	// AllowedOrigins sets the allowed values of the Origin header for
	// HTTP/JSON requests to an OTLP receiver. An origin may contain a
	// wildcard (*) to replace 0 or more characters (e.g.,
	// "http://*.domain.com", or "*" to allow any origin).
	AllowedOrigins []string `mapstructure:"allowed_origins"`

	// AllowedHeaders sets what headers will be allowed in CORS requests.
	// The Accept, Accept-Language, Content-Type, and Content-Language
	// headers are implicitly allowed. If no headers are listed,
	// X-Requested-With will also be accepted by default. Include "*" to
	// allow any request header.
	AllowedHeaders []string `mapstructure:"allowed_headers,omitempty"`

	// MaxAge sets the value of the Access-Control-Max-Age response header.
	// Set it to the number of seconds that browsers should cache a CORS
	// preflight response for.
	MaxAge int `mapstructure:"max_age,omitempty"`
}

func authInterceptor(next http.Handler, authenticate configauth.AuthenticateFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := authenticate(r.Context(), r.Header)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type compressRoundTripper struct {
	RoundTripper    http.RoundTripper
	compressionType CompressionType
	writer          func(*bytes.Buffer) (io.WriteCloser, error)
}

func newCompressRoundTripper(rt http.RoundTripper, compressionType CompressionType) *compressRoundTripper {
	return &compressRoundTripper{
		RoundTripper:    rt,
		compressionType: compressionType,
		writer:          writerFactory(compressionType),
	}
}

// writerFactory defines writer field in CompressRoundTripper.
// The validity of input is already checked when NewCompressRoundTripper was called in confighttp,
func writerFactory(compressionType CompressionType) func(*bytes.Buffer) (io.WriteCloser, error) {
	switch compressionType {
	case CompressionGzip:
		return func(buf *bytes.Buffer) (io.WriteCloser, error) {
			return gzip.NewWriter(buf), nil
		}
	case CompressionSnappy:
		return func(buf *bytes.Buffer) (io.WriteCloser, error) {
			return snappy.NewBufferedWriter(buf), nil
		}
	case CompressionZstd:
		return func(buf *bytes.Buffer) (io.WriteCloser, error) {
			return zstd.NewWriter(buf)
		}
	case CompressionZlib, CompressionDeflate:
		return func(buf *bytes.Buffer) (io.WriteCloser, error) {
			return zlib.NewWriter(buf), nil
		}
	}
	return nil
}

func (r *compressRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(headerContentEncoding) != "" {
		// If the header already specifies a content encoding then skip compression
		// since we don't want to compress it again. This is a safeguard that normally
		// should not happen since CompressRoundTripper is not intended to be used
		// with http clients which already do their own compression.
		return r.RoundTripper.RoundTrip(req)
	}

	// Compress the body.
	buf := bytes.NewBuffer([]byte{})
	compressWriter, writerErr := r.writer(buf)
	if writerErr != nil {
		return nil, writerErr
	}
	_, copyErr := io.Copy(compressWriter, req.Body)
	closeErr := req.Body.Close()

	if err := compressWriter.Close(); err != nil {
		return nil, err
	}

	if copyErr != nil {
		return nil, copyErr
	}

	if closeErr != nil {
		return nil, closeErr
	}

	// Create a new request since the docs say that we cannot modify the "req"
	// (see https://golang.org/pkg/net/http/#RoundTripper).
	cReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), buf)
	if err != nil {
		return nil, err
	}

	// Clone the headers and add gzip encoding header.
	cReq.Header = req.Header.Clone()
	cReq.Header.Add(headerContentEncoding, string(r.compressionType))

	return r.RoundTripper.RoundTrip(cReq)
}

type errorHandler func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)

type decompressor struct {
	errorHandler
}

type decompressorOption func(d *decompressor)

func withErrorHandlerForDecompressor(e errorHandler) decompressorOption {
	return func(d *decompressor) {
		d.errorHandler = e
	}
}

// httpContentDecompressor offloads the task of handling compressed HTTP requests
// by identifying the compression format in the "Content-Encoding" header and re-writing
// request body so that the handlers further in the chain can work on decompressed data.
// It supports gzip and deflate/zlib compression.
func httpContentDecompressor(h http.Handler, opts ...decompressorOption) http.Handler {
	d := &decompressor{}
	for _, o := range opts {
		o(d)
	}
	if d.errorHandler == nil {
		d.errorHandler = defaultErrorHandler
	}
	return d.wrap(h)
}

func (d *decompressor) wrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newBody, err := newBodyReader(r)
		if err != nil {
			d.errorHandler(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		if newBody != nil {
			defer newBody.Close()
			// "Content-Encoding" header is removed to avoid decompressing twice
			// in case the next handler(s) have implemented a similar mechanism.
			r.Header.Del("Content-Encoding")
			// "Content-Length" is set to -1 as the size of the decompressed body is unknown.
			r.Header.Del("Content-Length")
			r.ContentLength = -1
			r.Body = newBody
		}
		h.ServeHTTP(w, r)
	})
}

func newBodyReader(r *http.Request) (io.ReadCloser, error) {
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return gr, nil
	case "deflate", "zlib":
		zr, err := zlib.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return zr, nil
	}
	return nil, nil
}

// defaultErrorHandler writes the error message in plain text.
func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, errMsg string, statusCode int) {
	http.Error(w, errMsg, statusCode)
}

func (ct *CompressionType) UnmarshalText(in []byte) error {
	switch typ := CompressionType(in); typ {
	case CompressionGzip,
		CompressionZlib,
		CompressionDeflate,
		CompressionSnappy,
		CompressionZstd,
		CompressionNone,
		CompressionEmpty:
		*ct = typ
		return nil
	default:
		return fmt.Errorf("unsupported compression type %q", typ)
	}
}
