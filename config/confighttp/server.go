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
	"go.opentelemetry.io/collector/config/confighttp/internal"
	"go.opentelemetry.io/collector/config/confighttp/internal/metadata"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

const defaultMaxRequestBodySize = 20 * 1024 * 1024 // 20MiB

// ServerConfig defines settings for creating an HTTP server.
type privateServerConfigFields struct {
	// deprecationWarnings records use of deprecated fields observed while
	// unmarshaling; ToServer logs them, as no logger is available here.
	deprecationWarnings []string
}

// NewDefaultServerConfig returns ServerConfig type object with default values.
// We encourage to use this function to create an object of ServerConfig.
func NewDefaultServerConfig() ServerConfig {
	netAddr := confignet.NewDefaultAddrConfig()
	// We typically want to create a TCP server and listen over a network.
	netAddr.Transport = confignet.TransportTypeTCP

	if metadata.PkgConfighttpPrioritizeNewKeepaliveFeatureGate.IsEnabled() {
		return ServerConfig{
			NetAddr:           netAddr,
			WriteTimeout:      30 * time.Second,
			ReadHeaderTimeout: 1 * time.Minute,
			Keepalive:         configoptional.Some(NewDefaultKeepaliveServerConfig()),
		}
	}

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

func (sc *ServerConfig) Unmarshal(conf *confmap.Conf) error {
	if metadata.PkgConfighttpPrioritizeNewKeepaliveFeatureGate.IsEnabled() {
		return sc.unmarshalPrioritizeKeepalive(conf)
	}
	return sc.unmarshalPrioritizeDeprecatedFields(conf)
}

// unmarshalPrioritizeKeepalive implements confmap.Unmarshaler. The keepalive settings can arrive
// through two representations (the deprecated flat fields and the 'keepalive'
// section) and two channels (programmatic changes to the struct and the
// configuration). unmarshalPrioritizeKeepalive resolves them by folding everything into the
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
func (sc *ServerConfig) unmarshalPrioritizeKeepalive(conf *confmap.Conf) error {
	// Step 1: decode the configuration. Deprecated keys overwrite their
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

	// Step 2: with the decoded values at hand, reject configurations mixing
	// both representations, and record uses of the deprecated keys for
	// ToServer to warn about. Values which are no-ops in the legacy logic (a
	// zero idle_timeout, or keep_alives_enabled: true) neither conflict with
	// the 'keepalive' section nor deserve a warning.
	var deprecated []string
	if conf.IsSet("idle_timeout") && sc.IdleTimeout != 0 {
		sc.Keepalive.GetOrInsertDefault().IdleTimeout = sc.IdleTimeout
		deprecated = append(deprecated, "'idle_timeout' is deprecated; use 'keepalive::idle_timeout' instead")
	}
	if conf.IsSet("keep_alives_enabled") && !sc.KeepAlivesEnabled {
		deprecated = append(deprecated, "'keep_alives_enabled' is deprecated; set 'keepalive::enabled' to false to disable keep-alives")
	}
	if keepaliveSet && len(deprecated) > 0 {
		return errors.New("confighttp.ServerConfig: cannot use deprecated keepalive fields (idle_timeout, keep_alives_enabled) alongside the 'keepalive' section; migrate to the 'keepalive' section")
	}
	sc.deprecationWarnings = deprecated

	// Step 3: fold the decoded 'keepalive' section into the deprecated
	// fields. Only keys present in the configuration are copied; the
	// deprecated fields keep supplying the values for the rest. Decoding
	// leaves Keepalive without a value only for 'keepalive::enabled: false',
	// so a present section fully determines whether keep-alives are on.
	if ka := sc.Keepalive.Get(); ka != nil {
		if conf.IsSet("idle_timeout") {
			ka.IdleTimeout = sc.IdleTimeout
			sc.IdleTimeout = 0
		}
	}
	sc.KeepAlivesEnabled = false

	return nil
}

// unmarshalPrioritizeDeprecatedFields implements confmap.Unmarshaler. The keepalive settings can arrive
// through two representations (the deprecated flat fields and the 'keepalive'
// section) and two channels (programmatic changes to the struct and the
// configuration). unmarshalPrioritizeDeprecatedFields resolves them by folding everything into the
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
func (sc *ServerConfig) unmarshalPrioritizeDeprecatedFields(conf *confmap.Conf) error {
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
