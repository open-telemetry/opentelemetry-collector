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
		Keepalive:         configoptional.Some(NewDefaultKeepaliveServerConfig()),
	}
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
