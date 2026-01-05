// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp/internal"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

const defaultMaxRequestBodySize = 20 * 1024 * 1024 // 20MiB

// ServerConfig defines settings for creating an HTTP server.
type ServerConfig struct {
	// Endpoint configures the listening address for the server.
	Endpoint string `mapstructure:"endpoint,omitempty"`

	// TLS struct exposes TLS client configuration.
	TLS configoptional.Optional[configtls.ServerConfig] `mapstructure:"tls"`

	// CORS configures the server for HTTP cross-origin resource sharing (CORS).
	CORS configoptional.Optional[CORSConfig] `mapstructure:"cors"`

	// Auth for this receiver
	Auth configoptional.Optional[AuthConfig] `mapstructure:"auth,omitempty"`

	// MaxRequestBodySize sets the maximum request body size in bytes. Default: 20MiB.
	MaxRequestBodySize int64 `mapstructure:"max_request_body_size,omitempty"`

	// IncludeMetadata propagates the client metadata from the incoming requests to the downstream consumers
	IncludeMetadata bool `mapstructure:"include_metadata,omitempty"`

	// Additional headers attached to each HTTP response sent to the client.
	// Header values are opaque since they may be sensitive.
	ResponseHeaders configopaque.MapList `mapstructure:"response_headers,omitempty"`

	// CompressionAlgorithms configures the list of compression algorithms the server can accept. Default: ["", "gzip", "zstd", "zlib", "snappy", "deflate"]
	CompressionAlgorithms []string `mapstructure:"compression_algorithms,omitempty"`

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. A zero or negative value means
	// there will be no timeout.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration `mapstructure:"read_timeout,omitempty"`

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

	// Middlewares are used to add custom functionality to the HTTP server.
	// Middleware handlers are called in the order they appear in this list,
	// with the first middleware becoming the outermost handler.
	Middlewares []configmiddleware.Config `mapstructure:"middlewares,omitempty"`

	// KeepAlivesEnabled controls whether HTTP keep-alives are enabled.
	// By default, keep-alives are always enabled. Only very resource-constrained environments should disable them.
	KeepAlivesEnabled bool `mapstructure:"keep_alives_enabled,omitempty"`
}

// NewDefaultServerConfig returns ServerConfig type object with default values.
// We encourage to use this function to create an object of ServerConfig.
func NewDefaultServerConfig() ServerConfig {
	return ServerConfig{
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 1 * time.Minute,
		IdleTimeout:       1 * time.Minute,
		KeepAlivesEnabled: true,
	}
}

type AuthConfig struct {
	// Auth for this receiver.
	configauth.Config `mapstructure:",squash"`

	// RequestParameters is a list of parameters that should be extracted from the request and added to the context.
	// When a parameter is found in both the query string and the header, the value from the query string will be used.
	RequestParameters []string `mapstructure:"request_params,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// ToListener creates a net.Listener.
func (sc *ServerConfig) ToListener(ctx context.Context) (net.Listener, error) {
	listener, err := net.Listen("tcp", sc.Endpoint)
	if err != nil {
		return nil, err
	}

	if sc.TLS.HasValue() {
		var tlsCfg *tls.Config
		tlsCfg, err = sc.TLS.Get().LoadTLSConfig(ctx)
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
//
// To allow the configuration to reference middleware or authentication extensions,
// the `extensions` argument should be the output of `host.GetExtensions()`.
// It may also be `nil` in tests where no such extension is expected to be used.
func (sc *ServerConfig) ToServer(ctx context.Context, extensions map[component.ID]component.Component, settings component.TelemetrySettings, handler http.Handler, opts ...ToServerOption) (*http.Server, error) {
	serverOpts := &toServerOptions{}
	serverOpts.Apply(opts...)

	if sc.MaxRequestBodySize <= 0 {
		sc.MaxRequestBodySize = defaultMaxRequestBodySize
	}

	if sc.CompressionAlgorithms == nil {
		sc.CompressionAlgorithms = defaultCompressionAlgorithms()
	}

	// Apply middlewares in reverse order so they execute in
	// forward order.  The first middleware runs after
	// decompression, below, preceded by Auth, CORS, etc.
	if len(sc.Middlewares) > 0 && extensions == nil {
		return nil, errors.New("middlewares were configured but this component or its host does not support extensions")
	}
	for i := len(sc.Middlewares) - 1; i >= 0; i-- {
		wrapper, err := sc.Middlewares[i].GetHTTPServerHandler(ctx, extensions)
		// If we failed to get the middleware
		if err != nil {
			return nil, err
		}
		handler, err = wrapper(handler)
		// If we failed to construct a wrapper
		if err != nil {
			return nil, err
		}
	}

	handler = httpContentDecompressor(
		handler,
		sc.MaxRequestBodySize,
		serverOpts.ErrHandler,
		sc.CompressionAlgorithms,
		serverOpts.Decoders,
	)

	if sc.MaxRequestBodySize > 0 {
		handler = maxRequestBodySizeInterceptor(handler, sc.MaxRequestBodySize)
	}

	if sc.Auth.HasValue() {
		if extensions == nil {
			return nil, errors.New("authentication was configured but this component or its host does not support extensions")
		}

		auth := sc.Auth.Get()
		server, err := auth.GetServerAuthenticator(ctx, extensions)
		if err != nil {
			return nil, err
		}

		handler = authInterceptor(handler, server, auth.RequestParameters, serverOpts)
	}

	if sc.CORS.HasValue() && len(sc.CORS.Get().AllowedOrigins) > 0 {
		corsConfig := sc.CORS.Get()
		co := cors.Options{
			AllowedOrigins:   corsConfig.AllowedOrigins,
			AllowCredentials: true,
			AllowedHeaders:   corsConfig.AllowedHeaders,
			MaxAge:           corsConfig.MaxAge,
		}
		handler = cors.New(co).Handler(handler)
	}
	if sc.CORS.HasValue() && len(sc.CORS.Get().AllowedOrigins) == 0 && len(sc.CORS.Get().AllowedHeaders) > 0 {
		settings.Logger.Warn("The CORS configuration specifies allowed headers but no allowed origins, and is therefore ignored.")
	}

	if sc.ResponseHeaders != nil {
		handler = responseHeadersHandler(handler, sc.ResponseHeaders)
	}

	otelOpts := append(
		[]otelhttp.Option{
			otelhttp.WithTracerProvider(settings.TracerProvider),
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				// https://opentelemetry.io/docs/specs/semconv/http/http-spans/#name:
				//
				//   "HTTP span names SHOULD be {method} {target} if there is a (low-cardinality) target available.
				//   If there is no (low-cardinality) {target} available, HTTP span names SHOULD be {method}.
				//   ...
				//   Instrumentation MUST NOT default to using URI path as a {target}."
				//
				if r.Pattern != "" {
					return r.Method + " " + r.Pattern
				}
				return r.Method
			}),
			otelhttp.WithMeterProvider(settings.MeterProvider),
		},
		serverOpts.OtelhttpOpts...)

	// Enable OpenTelemetry observability plugin.
	handler = otelhttp.NewHandler(handler, "", otelOpts...)

	// wrap the current handler in an interceptor that will add client.Info to the request's context
	handler = &clientInfoHandler{
		next:            handler,
		includeMetadata: sc.IncludeMetadata,
	}

	errorLog, err := zap.NewStdLogAt(settings.Logger, zapcore.ErrorLevel)
	if err != nil {
		return nil, err // If an error occurs while creating the logger, return nil and the error
	}

	server := &http.Server{
		Handler:           handler,
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		WriteTimeout:      sc.WriteTimeout,
		IdleTimeout:       sc.IdleTimeout,
		ErrorLog:          errorLog,
	}

	// Set keep-alives enabled/disabled
	server.SetKeepAlivesEnabled(sc.KeepAlivesEnabled)

	return server, err
}

func responseHeadersHandler(handler http.Handler, headers configopaque.MapList) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		for k, v := range headers.Iter {
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
	AllowedOrigins []string `mapstructure:"allowed_origins,omitempty"`

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
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultCORSConfig creates a default cross-origin resource sharing (CORS) configuration.
func NewDefaultCORSConfig() CORSConfig {
	return CORSConfig{}
}

func authInterceptor(next http.Handler, server extensionauth.Server, requestParams []string, serverOpts *internal.ToServerOptions) http.Handler {
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
			if serverOpts.ErrHandler != nil {
				serverOpts.ErrHandler(w, r, err.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			}

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
