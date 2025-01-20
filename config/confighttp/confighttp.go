// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"golang.org/x/net/http2"
	"golang.org/x/net/publicsuffix"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/confighttp/internal"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension/auth"
)

const (
	headerContentEncoding     = "Content-Encoding"
	defaultMaxRequestBodySize = 20 * 1024 * 1024 // 20MiB
)

var defaultCompressionAlgorithms = []string{"", "gzip", "zstd", "zlib", "snappy", "deflate", "lz4"}

// ClientConfig defines settings for creating an HTTP client.
type ClientConfig struct {
	// The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
	Endpoint string `mapstructure:"endpoint"`

	// ProxyURL setting for the collector
	ProxyURL string `mapstructure:"proxy_url"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.ClientConfig `mapstructure:"tls"`

	// ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
	// Default is 0.
	ReadBufferSize int `mapstructure:"read_buffer_size"`

	// WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
	// Default is 0.
	WriteBufferSize int `mapstructure:"write_buffer_size"`

	// Timeout parameter configures `http.Client.Timeout`.
	// Default is 0 (unlimited).
	Timeout time.Duration `mapstructure:"timeout"`

	// Additional headers attached to each HTTP request sent by the client.
	// Existing header values are overwritten if collision happens.
	// Header values are opaque since they may be sensitive.
	Headers map[string]configopaque.String `mapstructure:"headers"`

	// Auth configuration for outgoing HTTP calls.
	Auth *configauth.Authentication `mapstructure:"auth"`

	// The compression key for supported compression types within collector.
	Compression configcompression.Type `mapstructure:"compression"`

	// Advanced configuration options for the Compression
	CompressionParams configcompression.CompressionParams `mapstructure:"compression_params"`

	// MaxIdleConns is used to set a limit to the maximum idle HTTP connections the client can keep open.
	// By default, it is set to 100.
	MaxIdleConns *int `mapstructure:"max_idle_conns"`

	// MaxIdleConnsPerHost is used to set a limit to the maximum idle HTTP connections the host can keep open.
	// By default, it is set to [http.DefaultTransport.MaxIdleConnsPerHost].
	MaxIdleConnsPerHost *int `mapstructure:"max_idle_conns_per_host"`

	// MaxConnsPerHost limits the total number of connections per host, including connections in the dialing,
	// active, and idle states.
	// By default, it is set to [http.DefaultTransport.MaxConnsPerHost].
	MaxConnsPerHost *int `mapstructure:"max_conns_per_host"`

	// IdleConnTimeout is the maximum amount of time a connection will remain open before closing itself.
	// By default, it is set to [http.DefaultTransport.IdleConnTimeout]
	IdleConnTimeout *time.Duration `mapstructure:"idle_conn_timeout"`

	// DisableKeepAlives, if true, disables HTTP keep-alives and will only use the connection to the server
	// for a single HTTP request.
	//
	// WARNING: enabling this option can result in significant overhead establishing a new HTTP(S)
	// connection for every request. Before enabling this option please consider whether changes
	// to idle connection settings can achieve your goal.
	DisableKeepAlives bool `mapstructure:"disable_keep_alives"`

	// This is needed in case you run into
	// https://github.com/golang/go/issues/59690
	// https://github.com/golang/go/issues/36026
	// HTTP2ReadIdleTimeout if the connection has been idle for the configured value send a ping frame for health check
	// 0s means no health check will be performed.
	HTTP2ReadIdleTimeout time.Duration `mapstructure:"http2_read_idle_timeout"`
	// HTTP2PingTimeout if there's no response to the ping within the configured value, the connection will be closed.
	// If not set or set to 0, it defaults to 15s.
	HTTP2PingTimeout time.Duration `mapstructure:"http2_ping_timeout"`
	// Cookies configures the cookie management of the HTTP client.
	Cookies *CookiesConfig `mapstructure:"cookies"`
}

// CookiesConfig defines the configuration of the HTTP client regarding cookies served by the server.
type CookiesConfig struct {
	// Enabled if true, cookies from HTTP responses will be reused in further HTTP requests with the same server.
	Enabled bool `mapstructure:"enabled"`
}

// NewDefaultClientConfig returns ClientConfig type object with
// the default values of 'MaxIdleConns' and 'IdleConnTimeout', as well as [http.DefaultTransport] values.
// Other config options are not added as they are initialized with 'zero value' by GoLang as default.
// We encourage to use this function to create an object of ClientConfig.
func NewDefaultClientConfig() ClientConfig {
	// The default values are taken from the values of 'DefaultTransport' of 'http' package.
	defaultTransport := http.DefaultTransport.(*http.Transport)

	return ClientConfig{
		ReadBufferSize:      defaultTransport.ReadBufferSize,
		WriteBufferSize:     defaultTransport.WriteBufferSize,
		Headers:             map[string]configopaque.String{},
		MaxIdleConns:        &defaultTransport.MaxIdleConns,
		MaxIdleConnsPerHost: &defaultTransport.MaxIdleConnsPerHost,
		MaxConnsPerHost:     &defaultTransport.MaxConnsPerHost,
		IdleConnTimeout:     &defaultTransport.IdleConnTimeout,
	}
}

func (hcs *ClientConfig) Validate() error {
	if hcs.Compression.IsCompressed() {
		if err := hcs.Compression.ValidateParams(hcs.CompressionParams); err != nil {
			return err
		}
	}
	return nil
}

// ToClient creates an HTTP client.
func (hcs *ClientConfig) ToClient(ctx context.Context, host component.Host, settings component.TelemetrySettings) (*http.Client, error) {
	tlsCfg, err := hcs.TLSSetting.LoadTLSConfig(ctx)
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

	// Setting the Proxy URL
	if hcs.ProxyURL != "" {
		proxyURL, parseErr := url.ParseRequestURI(hcs.ProxyURL)
		if parseErr != nil {
			return nil, parseErr
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	transport.DisableKeepAlives = hcs.DisableKeepAlives

	if hcs.HTTP2ReadIdleTimeout > 0 {
		transport2, transportErr := http2.ConfigureTransports(transport)
		if transportErr != nil {
			return nil, fmt.Errorf("failed to configure http2 transport: %w", transportErr)
		}
		transport2.ReadIdleTimeout = hcs.HTTP2ReadIdleTimeout
		transport2.PingTimeout = hcs.HTTP2PingTimeout
	}

	clientTransport := (http.RoundTripper)(transport)

	// The Auth RoundTripper should always be the innermost to ensure that
	// request signing-based auth mechanisms operate after compression
	// and header middleware modifies the request
	if hcs.Auth != nil {
		ext := host.GetExtensions()
		if ext == nil {
			return nil, errors.New("extensions configuration not found")
		}

		httpCustomAuthRoundTripper, aerr := hcs.Auth.GetClientAuthenticator(ctx, ext)
		if aerr != nil {
			return nil, aerr
		}

		clientTransport, err = httpCustomAuthRoundTripper.RoundTripper(clientTransport)
		if err != nil {
			return nil, err
		}
	}

	if len(hcs.Headers) > 0 {
		clientTransport = &headerRoundTripper{
			transport: clientTransport,
			headers:   hcs.Headers,
		}
	}

	// Compress the body using specified compression methods if non-empty string is provided.
	// Supporting gzip, zlib, deflate, snappy, and zstd; none is treated as uncompressed.
	if hcs.Compression.IsCompressed() {
		// If the compression level is not set, use the default level.
		if hcs.CompressionParams.Level == 0 {
			hcs.CompressionParams.Level = configcompression.DefaultCompressionLevel
		}
		clientTransport, err = newCompressRoundTripper(clientTransport, hcs.Compression, hcs.CompressionParams)
		if err != nil {
			return nil, err
		}
	}

	otelOpts := []otelhttp.Option{
		otelhttp.WithTracerProvider(settings.TracerProvider),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		otelhttp.WithMeterProvider(getLeveledMeterProvider(settings)),
	}
	// wrapping http transport with otelhttp transport to enable otel instrumentation
	if settings.TracerProvider != nil && settings.MeterProvider != nil {
		clientTransport = otelhttp.NewTransport(clientTransport, otelOpts...)
	}

	var jar http.CookieJar
	if hcs.Cookies != nil && hcs.Cookies.Enabled {
		jar, err = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, err
		}
	}

	return &http.Client{
		Transport: clientTransport,
		Timeout:   hcs.Timeout,
		Jar:       jar,
	}, nil
}

// Custom RoundTripper that adds headers.
type headerRoundTripper struct {
	transport http.RoundTripper
	headers   map[string]configopaque.String
}

// RoundTrip is a custom RoundTripper that adds headers to the request.
func (interceptor *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Set Host header if provided
	hostHeader, found := interceptor.headers["Host"]
	if found && hostHeader != "" {
		// `Host` field should be set to override default `Host` header value which is Endpoint
		req.Host = string(hostHeader)
	}
	for k, v := range interceptor.headers {
		req.Header.Set(k, string(v))
	}

	// Send the request to next transport.
	return interceptor.transport.RoundTrip(req)
}

// ServerConfig defines settings for creating an HTTP server.
type ServerConfig struct {
	// Endpoint configures the listening address for the server.
	Endpoint string `mapstructure:"endpoint"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *configtls.ServerConfig `mapstructure:"tls"`

	// CORS configures the server for HTTP cross-origin resource sharing (CORS).
	CORS *CORSConfig `mapstructure:"cors"`

	// Auth for this receiver
	Auth *AuthConfig `mapstructure:"auth"`

	// MaxRequestBodySize sets the maximum request body size in bytes. Default: 20MiB.
	MaxRequestBodySize int64 `mapstructure:"max_request_body_size"`

	// IncludeMetadata propagates the client metadata from the incoming requests to the downstream consumers
	IncludeMetadata bool `mapstructure:"include_metadata"`

	// Additional headers attached to each HTTP response sent to the client.
	// Header values are opaque since they may be sensitive.
	ResponseHeaders map[string]configopaque.String `mapstructure:"response_headers"`

	// CompressionAlgorithms configures the list of compression algorithms the server can accept. Default: ["", "gzip", "zstd", "zlib", "snappy", "deflate"]
	CompressionAlgorithms []string `mapstructure:"compression_algorithms"`

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. A zero or negative value means
	// there will be no timeout.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the Handler can decide what
	// is considered too slow for the body. If ReadHeaderTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	// A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

// NewDefaultServerConfig returns ServerConfig type object with default values.
// We encourage to use this function to create an object of ServerConfig.
func NewDefaultServerConfig() ServerConfig {
	tlsDefaultServerConfig := configtls.NewDefaultServerConfig()
	return ServerConfig{
		ResponseHeaders:   map[string]configopaque.String{},
		TLSSetting:        &tlsDefaultServerConfig,
		CORS:              NewDefaultCORSConfig(),
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 1 * time.Minute,
		IdleTimeout:       1 * time.Minute,
	}
}

type AuthConfig struct {
	// Auth for this receiver.
	configauth.Authentication `mapstructure:",squash"`

	// RequestParameters is a list of parameters that should be extracted from the request and added to the context.
	// When a parameter is found in both the query string and the header, the value from the query string will be used.
	RequestParameters []string `mapstructure:"request_params"`
}

// ToListener creates a net.Listener.
func (hss *ServerConfig) ToListener(ctx context.Context) (net.Listener, error) {
	listener, err := net.Listen("tcp", hss.Endpoint)
	if err != nil {
		return nil, err
	}

	if hss.TLSSetting != nil {
		var tlsCfg *tls.Config
		tlsCfg, err = hss.TLSSetting.LoadTLSConfig(ctx)
		if err != nil {
			return nil, err
		}
		tlsCfg.NextProtos = []string{http2.NextProtoTLS, "http/1.1"}
		listener = tls.NewListener(listener, tlsCfg)
	}

	return listener, nil
}

// toServerOptions has options that change the behavior of the HTTP server
// returned by ServerConfig.ToServer().
type toServerOptions = internal.ToServerOptions

// ToServerOption is an option to change the behavior of the HTTP server
// returned by ServerConfig.ToServer().
type ToServerOption = internal.ToServerOption

// WithErrorHandler overrides the HTTP error handler that gets invoked
// when there is a failure inside httpContentDecompressor.
func WithErrorHandler(e func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)) ToServerOption {
	return internal.ToServerOptionFunc(func(opts *toServerOptions) {
		opts.ErrHandler = e
	})
}

// WithDecoder provides support for additional decoders to be configured
// by the caller.
func WithDecoder(key string, dec func(body io.ReadCloser) (io.ReadCloser, error)) ToServerOption {
	return internal.ToServerOptionFunc(func(opts *toServerOptions) {
		if opts.Decoders == nil {
			opts.Decoders = map[string]func(body io.ReadCloser) (io.ReadCloser, error){}
		}
		opts.Decoders[key] = dec
	})
}

// ToServer creates an http.Server from settings object.
func (hss *ServerConfig) ToServer(_ context.Context, host component.Host, settings component.TelemetrySettings, handler http.Handler, opts ...ToServerOption) (*http.Server, error) {
	serverOpts := &toServerOptions{}
	serverOpts.Apply(opts...)

	if hss.MaxRequestBodySize <= 0 {
		hss.MaxRequestBodySize = defaultMaxRequestBodySize
	}

	if hss.CompressionAlgorithms == nil {
		hss.CompressionAlgorithms = defaultCompressionAlgorithms
	}

	handler = httpContentDecompressor(
		handler,
		hss.MaxRequestBodySize,
		serverOpts.ErrHandler,
		hss.CompressionAlgorithms,
		serverOpts.Decoders,
	)

	if hss.MaxRequestBodySize > 0 {
		handler = maxRequestBodySizeInterceptor(handler, hss.MaxRequestBodySize)
	}

	if hss.Auth != nil {
		server, err := hss.Auth.GetServerAuthenticator(context.Background(), host.GetExtensions())
		if err != nil {
			return nil, err
		}

		handler = authInterceptor(handler, server, hss.Auth.RequestParameters)
	}

	if hss.CORS != nil && len(hss.CORS.AllowedOrigins) > 0 {
		co := cors.Options{
			AllowedOrigins:   hss.CORS.AllowedOrigins,
			AllowCredentials: true,
			AllowedHeaders:   hss.CORS.AllowedHeaders,
			MaxAge:           hss.CORS.MaxAge,
		}
		handler = cors.New(co).Handler(handler)
	}
	if hss.CORS != nil && len(hss.CORS.AllowedOrigins) == 0 && len(hss.CORS.AllowedHeaders) > 0 {
		settings.Logger.Warn("The CORS configuration specifies allowed headers but no allowed origins, and is therefore ignored.")
	}

	if hss.ResponseHeaders != nil {
		handler = responseHeadersHandler(handler, hss.ResponseHeaders)
	}

	otelOpts := append(
		[]otelhttp.Option{
			otelhttp.WithTracerProvider(settings.TracerProvider),
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return r.URL.Path
			}),
			otelhttp.WithMeterProvider(getLeveledMeterProvider(settings)),
		},
		serverOpts.OtelhttpOpts...)

	// Enable OpenTelemetry observability plugin.
	// TODO: Consider to use component ID string as prefix for all the operations.
	handler = otelhttp.NewHandler(handler, "", otelOpts...)

	// wrap the current handler in an interceptor that will add client.Info to the request's context
	handler = &clientInfoHandler{
		next:            handler,
		includeMetadata: hss.IncludeMetadata,
	}

	server := &http.Server{
		Handler:           handler,
		ReadTimeout:       hss.ReadTimeout,
		ReadHeaderTimeout: hss.ReadHeaderTimeout,
		WriteTimeout:      hss.WriteTimeout,
		IdleTimeout:       hss.IdleTimeout,
	}

	return server, nil
}

func responseHeadersHandler(handler http.Handler, headers map[string]configopaque.String) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		for k, v := range headers {
			h.Set(k, string(v))
		}

		handler.ServeHTTP(w, r)
	})
}

// CORSConfig configures a receiver for HTTP cross-origin resource sharing (CORS).
// See the underlying https://github.com/rs/cors package for details.
type CORSConfig struct {
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
	AllowedHeaders []string `mapstructure:"allowed_headers"`

	// MaxAge sets the value of the Access-Control-Max-Age response header.
	// Set it to the number of seconds that browsers should cache a CORS
	// preflight response for.
	MaxAge int `mapstructure:"max_age"`
}

// NewDefaultCORSConfig creates a default cross-origin resource sharing (CORS) configuration.
func NewDefaultCORSConfig() *CORSConfig {
	return &CORSConfig{}
}

func authInterceptor(next http.Handler, server auth.Server, requestParams []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sources := r.Header
		query := r.URL.Query()
		for _, param := range requestParams {
			if val, ok := query[param]; ok {
				sources[param] = val
			}
		}
		ctx, err := server.Authenticate(r.Context(), sources)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func maxRequestBodySizeInterceptor(next http.Handler, maxRecvSize int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRecvSize)
		next.ServeHTTP(w, r)
	})
}

func getLeveledMeterProvider(settings component.TelemetrySettings) metric.MeterProvider {
	if configtelemetry.LevelDetailed <= settings.MetricsLevel {
		return settings.MeterProvider
	}
	return noop.MeterProvider{}
}
