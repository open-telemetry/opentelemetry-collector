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
	"slices"
	"strings"
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
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

const defaultMaxRequestBodySize = 20 * 1024 * 1024 // 20MiB

// ServerConfig defines settings for creating an HTTP server.
type ServerConfig struct {
	// NetAddr holds configuration for the network listener.
	//
	// Transport defaults to "tcp" if unspecified, and only
	// "tcp", "tcp4", "tcp6", and "unix" are valid options.
	NetAddr confignet.AddrConfig `mapstructure:",squash"`

	// TLS struct exposes TLS server configuration.
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

	// Middlewares are used to add custom functionality to the HTTP server.
	// Middleware handlers are called in the order they appear in this list,
	// with the first middleware becoming the outermost handler.
	Middlewares []configmiddleware.Config `mapstructure:"middlewares,omitempty"`

	// Keepalive controls HTTP keep-alives.
	// By default, keep-alives are always enabled. Only very resource-constrained environments should disable them.
	// Unmarshal folds this section into the deprecated fields below, which
	// remain the source of truth during their deprecation window, and always
	// resets it to None. A value visible to ToServer was therefore set
	// programmatically after unmarshaling (or the config was never unmarshaled)
	// and takes precedence over the deprecated fields.
	Keepalive configoptional.Optional[KeepaliveServerConfig] `mapstructure:"keepalive,omitempty"`

	// Deprecated: use Keepalive.IdleTimeout instead.
	IdleTimeout time.Duration `mapstructure:"idle_timeout,omitempty"`
	// Deprecated: set 'keepalive::enabled' to false to disable keep-alives.
	KeepAlivesEnabled bool `mapstructure:"keep_alives_enabled,omitempty"`

	// deprecationWarnings records use of deprecated fields observed while
	// unmarshaling; ToServer logs them, as no logger is available here.
	deprecationWarnings []string

	// prevent unkeyed literal initialization
	_ struct{}
}

type KeepaliveServerConfig struct {
	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultKeepaliveServerConfig returns a KeepaliveServerConfig with the same
// defaults that NewDefaultServerConfig sets on the corresponding deprecated
// fields.
func NewDefaultKeepaliveServerConfig() KeepaliveServerConfig {
	return KeepaliveServerConfig{
		IdleTimeout: 1 * time.Minute,
	}
}

// NewDefaultServerConfig returns ServerConfig type object with default values.
// We encourage to use this function to create an object of ServerConfig.
func NewDefaultServerConfig() ServerConfig {
	netAddr := confignet.NewDefaultAddrConfig()
	// We typically want to create a TCP server and listen over a network.
	netAddr.Transport = confignet.TransportTypeTCP

	return ServerConfig{
		NetAddr:           netAddr,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 1 * time.Minute,
		// The deprecated flat fields keep carrying the defaults so that
		// configurations and code which still use them behave as before.
		// Keepalive stays None; see its documentation.
		IdleTimeout:       1 * time.Minute,
		KeepAlivesEnabled: true,
	}
}

var _ confmap.Unmarshaler = (*ServerConfig)(nil)

// Unmarshal implements confmap.Unmarshaler. The keepalive settings can arrive
// through two representations (the deprecated flat fields and the 'keepalive'
// section) and two channels (programmatic changes to the struct and the
// configuration). Unmarshal resolves them by folding everything into the
// deprecated fields, in order of increasing precedence:
//
//  1. programmatic values already on the struct, in either representation;
//  2. deprecated keys present in the configuration;
//  3. the 'keepalive' section present in the configuration.
//
// Mixing 2 and 3 is rejected, so their relative precedence never matters in
// practice. Deprecated keys in the configuration are also recorded as warnings
// for ToServer to log. Only keys present in the configuration count for the
// error and the warnings: programmatic values are neither deprecated usage nor
// a conflict.
//
// The deprecated fields end up holding the effective settings and remain the
// sole source of truth for ToServer during their deprecation window; Keepalive
// is always left as None.
func (sc *ServerConfig) Unmarshal(conf *confmap.Conf) error {
	// Step 1: fold a programmatically set Keepalive into the deprecated
	// fields. This must precede decoding so that the configuration overrides
	// it. A present value can only mean keep-alives enabled with these
	// settings.
	if ka := sc.Keepalive.Get(); ka != nil {
		sc.IdleTimeout = ka.IdleTimeout
		sc.KeepAlivesEnabled = true
	}

	// Step 2: decode the configuration. Deprecated keys overwrite their
	// fields directly; the 'keepalive' section decodes into Keepalive and is
	// folded in step 4. WithIgnoreUnused is needed because ServerConfig is
	// commonly squash-embedded into component configs, in which case conf
	// also holds the parent's sibling fields.
	if err := conf.Unmarshal(sc, confmap.WithIgnoreUnused()); err != nil {
		return err
	}

	// A null 'keepalive' key carries no settings, but decodes as an enabled
	// section. Marshaling produces it for an unset Keepalive, so treat it as
	// unset to keep marshaled configurations loadable.
	keepaliveSet := conf.IsSet("keepalive") && conf.Get("keepalive") != nil

	// Step 3: with the decoded values at hand, reject configurations mixing
	// both representations, and record uses of the deprecated keys for
	// ToServer to warn about. Values which are no-ops in the legacy logic (a
	// zero idle_timeout, or keep_alives_enabled: true) neither conflict with
	// the 'keepalive' section nor deserve a warning.
	var deprecated []string
	if conf.IsSet("idle_timeout") && sc.IdleTimeout != 0 {
		deprecated = append(deprecated, "'idle_timeout' is deprecated; use 'keepalive::idle_timeout' instead")
	}
	if conf.IsSet("keep_alives_enabled") && !sc.KeepAlivesEnabled {
		deprecated = append(deprecated, "'keep_alives_enabled' is deprecated; set 'keepalive::enabled' to false to disable keep-alives")
	}
	if keepaliveSet && len(deprecated) > 0 {
		return errors.New("confighttp.ServerConfig: cannot use deprecated keepalive fields (idle_timeout, keep_alives_enabled) alongside the 'keepalive' section; migrate to the 'keepalive' section")
	}
	sc.deprecationWarnings = deprecated

	// Step 4: fold the decoded 'keepalive' section into the deprecated
	// fields. Only keys present in the configuration are copied; the
	// deprecated fields keep supplying the values for the rest. Decoding
	// leaves Keepalive without a value only for 'keepalive::enabled: false',
	// so a present section fully determines whether keep-alives are on.
	if keepaliveSet {
		if ka := sc.Keepalive.Get(); ka != nil {
			if conf.IsSet("keepalive::idle_timeout") {
				sc.IdleTimeout = ka.IdleTimeout
			}
		}
		sc.KeepAlivesEnabled = sc.Keepalive.HasValue()
	}

	// Step 5: the deprecated fields now hold the effective settings; restore
	// the invariant that Keepalive is None after unmarshaling (see the field
	// documentation).
	sc.Keepalive = configoptional.None[KeepaliveServerConfig]()
	return nil
}

type AuthConfig struct {
	// Auth for this receiver.
	Config configauth.Config `mapstructure:",squash"`

	// RequestParameters is a list of parameters that should be extracted from the request and added to the context.
	// When a parameter is found in both the query string and the header, the value from the query string will be used.
	RequestParameters []string `mapstructure:"request_params,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// ToListener creates a net.Listener.
func (sc *ServerConfig) ToListener(ctx context.Context) (net.Listener, error) {
	listener, err := sc.NetAddr.Listen(ctx)
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
	for _, warning := range sc.deprecationWarnings {
		settings.Logger.Warn(warning)
	}

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
	for _, m := range slices.Backward(sc.Middlewares) {
		wrapper, err := m.GetHTTPServerHandler(ctx, extensions)
		// If we failed to get the middleware
		if err != nil {
			return nil, err
		}
		handler, err = wrapper(ctx, handler)
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
		server, err := auth.Config.GetServerAuthenticator(ctx, extensions)
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
			ExposedHeaders:   corsConfig.ExposedHeaders,
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
				//
				//   The {method} MUST be {http.request.method} if the method represents the original method known
				//   to the instrumentation. In other cases (when {http.request.method} is set to _OTHER),
				//   {method} MUST be HTTP.
				//
				//   Instrumentation MUST NOT default to using URI path as a {target}."
				//
				method := standardizeHTTPMethod(r.Method, "HTTP")
				if r.Pattern != "" {
					return method + " " + r.Pattern
				}
				return method
			}),
			otelhttp.WithMeterProvider(settings.MeterProvider),
		},
		serverOpts.OtelhttpOpts...,
	)

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

	keepAlivesEnabled := true
	var idleTimeout time.Duration
	if kaCfg := sc.Keepalive.Get(); kaCfg != nil {
		// Unmarshal always leaves Keepalive at None, so a value here was set
		// programmatically afterwards and takes precedence.
		idleTimeout = kaCfg.IdleTimeout
	} else {
		// Apply the deprecated flat fields exactly as the code before the
		// 'keepalive' section's introduction did; Unmarshal has already folded
		// the section into them.
		keepAlivesEnabled = sc.KeepAlivesEnabled
		idleTimeout = sc.IdleTimeout
	}

	server := &http.Server{
		Handler:           handler,
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		WriteTimeout:      sc.WriteTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          errorLog,
	}

	server.SetKeepAlivesEnabled(keepAlivesEnabled)

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

	// ExposedHeaders sets the value of the Access-Control-Expose-Headers response
	// header, indicating which headers are safe to expose to the API of a CORS response.
	ExposedHeaders []string `mapstructure:"exposed_headers,omitempty"`

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

// standardizeHTTPMethod returns an upper case HTTP method if well-known, otherwise unknown.
// Based on https://github.com/open-telemetry/opentelemetry-go-contrib/blob/1530d71edc6d40d0659187d069081b639ef1b394/instrumentation/github.com/emicklei/go-restful/otelrestful/internal/semconv/util.go#L119
func standardizeHTTPMethod(method, unknown string) string {
	method = strings.ToUpper(method)
	switch method {
	case http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace:
		return method
	}
	return unknown
}
