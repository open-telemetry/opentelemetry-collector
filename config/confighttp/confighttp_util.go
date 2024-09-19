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

	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/http2"
	"golang.org/x/net/publicsuffix"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/config/internal"
	"go.opentelemetry.io/collector/extension/auth"
)

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
	if hcs.ProxyUrl != "" {
		proxyURL, parseErr := url.ParseRequestURI(hcs.ProxyUrl)
		if parseErr != nil {
			return nil, parseErr
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	transport.DisableKeepAlives = hcs.DisableKeepAlives

	if hcs.Http2ReadIdleTimeout > 0 {
		transport2, transportErr := http2.ConfigureTransports(transport)
		if transportErr != nil {
			return nil, fmt.Errorf("failed to configure http2 transport: %w", transportErr)
		}
		transport2.ReadIdleTimeout = hcs.Http2ReadIdleTimeout
		transport2.PingTimeout = hcs.Http2PingTimeout
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
		clientTransport, err = newCompressRoundTripper(clientTransport, hcs.Compression)
		if err != nil {
			return nil, err
		}
	}

	otelOpts := []otelhttp.Option{
		otelhttp.WithTracerProvider(settings.TracerProvider),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	}
	if settings.MetricsLevel >= configtelemetry.LevelDetailed {
		otelOpts = append(otelOpts, otelhttp.WithMeterProvider(settings.MeterProvider))
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
type toServerOptions struct {
	errHandler func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)
	decoders   map[string]func(body io.ReadCloser) (io.ReadCloser, error)
}

// ToServerOption is an option to change the behavior of the HTTP server
// returned by ServerConfig.ToServer().
type ToServerOption func(opts *toServerOptions)

// WithErrorHandler overrides the HTTP error handler that gets invoked
// when there is a failure inside httpContentDecompressor.
func WithErrorHandler(e func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)) ToServerOption {
	return func(opts *toServerOptions) {
		opts.errHandler = e
	}
}

// WithDecoder provides support for additional decoders to be configured
// by the caller.
func WithDecoder(key string, dec func(body io.ReadCloser) (io.ReadCloser, error)) ToServerOption {
	return func(opts *toServerOptions) {
		if opts.decoders == nil {
			opts.decoders = map[string]func(body io.ReadCloser) (io.ReadCloser, error){}
		}
		opts.decoders[key] = dec
	}
}

// ToServer creates an http.Server from settings object.
func (hss *ServerConfig) ToServer(_ context.Context, host component.Host, settings component.TelemetrySettings, handler http.Handler, opts ...ToServerOption) (*http.Server, error) {
	internal.WarnOnUnspecifiedHost(settings.Logger, hss.Endpoint)

	serverOpts := &toServerOptions{}
	for _, o := range opts {
		o(serverOpts)
	}

	if hss.MaxRequestBodySize <= 0 {
		hss.MaxRequestBodySize = defaultMaxRequestBodySize
	}

	if hss.CompressionAlgorithms == nil {
		hss.CompressionAlgorithms = defaultCompressionAlgorithms
	}

	handler = httpContentDecompressor(handler, hss.MaxRequestBodySize, serverOpts.errHandler, hss.CompressionAlgorithms, serverOpts.decoders)

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

	if hss.Cors != nil && len(hss.Cors.AllowedOrigins) > 0 {
		co := cors.Options{
			AllowedOrigins:   hss.Cors.AllowedOrigins,
			AllowCredentials: true,
			AllowedHeaders:   hss.Cors.AllowedHeaders,
			MaxAge:           hss.Cors.MaxAge,
		}
		handler = cors.New(co).Handler(handler)
	}
	if hss.Cors != nil && len(hss.Cors.AllowedOrigins) == 0 && len(hss.Cors.AllowedHeaders) > 0 {
		settings.Logger.Warn("The CORS configuration specifies allowed headers but no allowed origins, and is therefore ignored.")
	}

	if hss.ResponseHeaders != nil {
		handler = responseHeadersHandler(handler, hss.ResponseHeaders)
	}

	otelOpts := []otelhttp.Option{
		otelhttp.WithTracerProvider(settings.TracerProvider),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.URL.Path
		}),
	}
	if settings.MetricsLevel >= configtelemetry.LevelDetailed {
		otelOpts = append(otelOpts, otelhttp.WithMeterProvider(settings.MeterProvider))
	}

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
