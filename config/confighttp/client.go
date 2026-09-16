// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/http2"
	"golang.org/x/net/publicsuffix"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
)

const (
	headerContentEncoding = "Content-Encoding"
)

type privateClientConfigFields struct {
	// deprecationWarnings records use of deprecated fields observed while
	// unmarshaling; ToClient logs them, as no logger is available here.
	deprecationWarnings []string
}

// CookiesConfig defines the configuration of the HTTP client regarding cookies served by the server.
type CookiesConfig struct {
	_ struct{}
}

// NewDefaultKeepaliveClientConfig returns a KeepaliveClientConfig with the same
// defaults that NewDefaultClientConfig sets on the corresponding deprecated
// fields, taken from [http.DefaultTransport].
func NewDefaultKeepaliveClientConfig() KeepaliveClientConfig {
	defaultTransport := http.DefaultTransport.(*http.Transport)

	return KeepaliveClientConfig{
		IdleConnTimeout: defaultTransport.IdleConnTimeout,
		MaxIdleConns:    defaultTransport.MaxIdleConns,
	}
}

// NewDefaultClientConfig returns ClientConfig type object with [http.DefaultTransport] values.
// Other config options are not added as they are initialized with 'zero value' by GoLang as default.
// We encourage to use this function to create an object of ClientConfig.
func NewDefaultClientConfig() ClientConfig {
	return ClientConfig{
		Keepalive:         configoptional.Some(NewDefaultKeepaliveClientConfig()),
		ForceAttemptHTTP2: true,
	}
}

var _ confmap.Unmarshaler = (*ClientConfig)(nil)

// Unmarshal implements confmap.Unmarshaler.
func (cc *ClientConfig) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(cc, confmap.WithIgnoreUnused()); err != nil {
		return err
	}

	return nil
}

func (cc *ClientConfig) Validate() error {
	if cc.Compression.IsCompressed() {
		if err := cc.Compression.ValidateParams(cc.CompressionParams); err != nil {
			return err
		}
	}
	return nil
}

// ToClientOption is an option to change the behavior of the HTTP client
// returned by ClientConfig.ToClient().
// There are currently no available options.
type ToClientOption interface {
	sealed()
}

// ToClient creates an HTTP client.
//
// To allow the configuration to reference middleware or authentication extensions,
// the `extensions` argument should be the output of `host.GetExtensions()`.
// It may also be `nil` in tests where no such extension is expected to be used.
func (cc *ClientConfig) ToClient(ctx context.Context, extensions map[component.ID]component.Component, settings component.TelemetrySettings, _ ...ToClientOption) (*http.Client, error) {
	for _, warning := range cc.deprecationWarnings {
		settings.Logger.Warn(warning)
	}

	tlsCfg, err := cc.TLS.LoadTLSConfig(ctx)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	if cc.ReadBufferSize > 0 {
		transport.ReadBufferSize = cc.ReadBufferSize
	}
	if cc.WriteBufferSize > 0 {
		transport.WriteBufferSize = cc.WriteBufferSize
	}

	if kaCfg := cc.Keepalive.Get(); kaCfg != nil {
		// Unmarshal always leaves Keepalive at None, so a value here was set
		// programmatically afterwards and takes precedence.
		transport.MaxIdleConns = kaCfg.MaxIdleConns
		transport.MaxIdleConnsPerHost = kaCfg.MaxIdleConnsPerHost
		transport.IdleConnTimeout = kaCfg.IdleConnTimeout
	}
	transport.MaxConnsPerHost = cc.MaxConnsPerHost
	transport.ForceAttemptHTTP2 = cc.ForceAttemptHTTP2
	// Setting the Proxy URL
	if cc.ProxyURL != "" {
		proxyURL, parseErr := url.ParseRequestURI(cc.ProxyURL)
		if parseErr != nil {
			return nil, parseErr
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	if cc.HTTP2ReadIdleTimeout > 0 {
		transport2, transportErr := http2.ConfigureTransports(transport) //nolint:staticcheck // SA1019
		if transportErr != nil {
			return nil, fmt.Errorf("failed to configure http2 transport: %w", transportErr)
		}
		transport2.ReadIdleTimeout = cc.HTTP2ReadIdleTimeout //nolint:staticcheck // SA1019
		transport2.PingTimeout = cc.HTTP2PingTimeout         //nolint:staticcheck // SA1019
	}

	clientTransport := http.RoundTripper(transport)

	// Apply middlewares in reverse order so they execute in
	// forward order. The first middleware runs after authentication.
	if len(cc.Middlewares) > 0 && extensions == nil {
		return nil, errors.New("middlewares were configured but this component or its host does not support extensions")
	}
	for _, m := range slices.Backward(cc.Middlewares) {
		getClient, rerr := m.GetHTTPClientRoundTripper(ctx, extensions)
		// If we failed to get the middleware
		if rerr != nil {
			return nil, rerr
		}
		clientTransport, rerr = getClient(ctx, clientTransport)
		// If we failed to construct a wrapper
		if rerr != nil {
			return nil, rerr
		}
	}

	// The Auth RoundTripper should always be the innermost to ensure that
	// request signing-based auth mechanisms operate after compression
	// and header middleware modifies the request
	if cc.Auth.HasValue() {
		if extensions == nil {
			return nil, errors.New("authentication was configured but this component or its host does not support extensions")
		}

		auth := cc.Auth.Get()
		httpCustomAuthRoundTripper, aerr := auth.GetHTTPClientAuthenticator(ctx, extensions)
		if aerr != nil {
			return nil, aerr
		}

		clientTransport, err = httpCustomAuthRoundTripper.RoundTripper(clientTransport)
		if err != nil {
			return nil, err
		}
	}

	if len(cc.Headers) > 0 {
		clientTransport = &headerRoundTripper{
			transport: clientTransport,
			headers:   cc.Headers,
		}
	}

	// Compress the body using specified compression methods if non-empty string is provided.
	// Supporting gzip, zlib, deflate, snappy, and zstd; none is treated as uncompressed.
	if cc.Compression.IsCompressed() {
		// If the compression level is not set, use the default level.
		if cc.CompressionParams.Level == 0 {
			cc.CompressionParams.Level = configcompression.DefaultCompressionLevel
		}
		clientTransport, err = newCompressRoundTripper(clientTransport, cc.Compression, cc.CompressionParams)
		if err != nil {
			return nil, err
		}
	}

	otelOpts := []otelhttp.Option{
		otelhttp.WithTracerProvider(settings.TracerProvider),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		otelhttp.WithMeterProvider(settings.MeterProvider),
	}
	// wrapping http transport with otelhttp transport to enable otel instrumentation
	if settings.TracerProvider != nil && settings.MeterProvider != nil {
		clientTransport = otelhttp.NewTransport(clientTransport, otelOpts...)
	}

	var jar http.CookieJar
	if cc.Cookies.HasValue() {
		jar, err = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, err
		}
	}

	return &http.Client{
		Transport: clientTransport,
		Timeout:   cc.Timeout,
		Jar:       jar,
	}, nil
}

// Custom RoundTripper that adds headers.
type headerRoundTripper struct {
	transport http.RoundTripper
	headers   configopaque.MapList
}

// RoundTrip is a custom RoundTripper that adds headers to the request.
func (interceptor *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Set Host header if provided
	hostHeader, found := interceptor.headers.Get("Host")
	if found && hostHeader != "" {
		// `Host` field should be set to override default `Host` header value which is Endpoint
		req.Host = string(hostHeader)
	}
	for k, v := range interceptor.headers.Iter {
		req.Header.Set(k, string(v))
	}

	// Send the request to next transport.
	return interceptor.transport.RoundTrip(req)
}
