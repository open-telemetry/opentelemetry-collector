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
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/http2"
	"golang.org/x/net/publicsuffix"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
)

const (
	headerContentEncoding = "Content-Encoding"
)

// ClientConfig defines settings for creating an HTTP client.
type ClientConfig struct {
	// The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
	Endpoint string `mapstructure:"endpoint,omitempty"`

	// ProxyURL setting for the collector
	ProxyURL string `mapstructure:"proxy_url,omitempty"`

	// TLS struct exposes TLS client configuration.
	TLS configtls.ClientConfig `mapstructure:"tls,omitempty"`

	// ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
	// Default is 0.
	ReadBufferSize int `mapstructure:"read_buffer_size,omitempty"`

	// WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
	// Default is 0.
	WriteBufferSize int `mapstructure:"write_buffer_size,omitempty"`

	// Timeout parameter configures `http.Client.Timeout`.
	// Default is 0 (unlimited).
	Timeout time.Duration `mapstructure:"timeout,omitempty"`

	// Additional headers attached to each HTTP request sent by the client.
	// Existing header values are overwritten if collision happens.
	// Header values are opaque since they may be sensitive.
	Headers configopaque.MapList `mapstructure:"headers,omitempty"`

	// Auth configuration for outgoing HTTP calls.
	Auth configoptional.Optional[configauth.Config] `mapstructure:"auth,omitempty"`

	// The compression key for supported compression types within collector.
	Compression configcompression.Type `mapstructure:"compression,omitempty"`

	// Advanced configuration options for the Compression
	CompressionParams configcompression.CompressionParams `mapstructure:"compression_params,omitempty"`

	// MaxIdleConns is used to set a limit to the maximum idle HTTP connections the client can keep open.
	// By default, it is set to 100. Zero means no limit.
	MaxIdleConns int `mapstructure:"max_idle_conns"`

	// MaxIdleConnsPerHost is used to set a limit to the maximum idle HTTP connections the host can keep open.
	// If zero, [net/http.DefaultMaxIdleConnsPerHost] is used.
	MaxIdleConnsPerHost int `mapstructure:"max_idle_conns_per_host,omitempty"`

	// MaxConnsPerHost limits the total number of connections per host, including connections in the dialing,
	// active, and idle states. Default is 0 (unlimited).
	MaxConnsPerHost int `mapstructure:"max_conns_per_host,omitempty"`

	// IdleConnTimeout is the maximum amount of time a connection will remain open before closing itself.
	// By default, it is set to 90 seconds.
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout"`

	// DisableKeepAlives, if true, disables HTTP keep-alives and will only use the connection to the server
	// for a single HTTP request.
	//
	// WARNING: enabling this option can result in significant overhead establishing a new HTTP(S)
	// connection for every request. Before enabling this option please consider whether changes
	// to idle connection settings can achieve your goal.
	DisableKeepAlives bool `mapstructure:"disable_keep_alives,omitempty"`

	// This is needed in case you run into
	// https://github.com/golang/go/issues/59690
	// https://github.com/golang/go/issues/36026
	// HTTP2ReadIdleTimeout if the connection has been idle for the configured value send a ping frame for health check
	// 0s means no health check will be performed.
	HTTP2ReadIdleTimeout time.Duration `mapstructure:"http2_read_idle_timeout,omitempty"`
	// HTTP2PingTimeout if there's no response to the ping within the configured value, the connection will be closed.
	// If not set or set to 0, it defaults to 15s.
	HTTP2PingTimeout time.Duration `mapstructure:"http2_ping_timeout,omitempty"`
	// Cookies configures the cookie management of the HTTP client.
	Cookies configoptional.Optional[CookiesConfig] `mapstructure:"cookies,omitempty"`

	// Enabling ForceAttemptHTTP2 forces the HTTP transport to use the HTTP/2 protocol.
	// By default, this is set to true.
	// NOTE: HTTP/2 does not support settings such as MaxConnsPerHost, MaxIdleConnsPerHost and MaxIdleConns.
	ForceAttemptHTTP2 bool `mapstructure:"force_attempt_http2,omitempty"`

	// Middlewares are used to add custom functionality to the HTTP client.
	// Middleware handlers are called in the order they appear in this list,
	// with the first middleware becoming the outermost handler.
	Middlewares []configmiddleware.Config `mapstructure:"middlewares,omitempty"`
}

// CookiesConfig defines the configuration of the HTTP client regarding cookies served by the server.
type CookiesConfig struct {
	_ struct{}
}

// NewDefaultClientConfig returns ClientConfig type object with
// the default values of 'MaxIdleConns' and 'IdleConnTimeout', as well as [http.DefaultTransport] values.
// Other config options are not added as they are initialized with 'zero value' by GoLang as default.
// We encourage to use this function to create an object of ClientConfig.
func NewDefaultClientConfig() ClientConfig {
	// The default values are taken from the values of 'DefaultTransport' of 'http' package.
	defaultTransport := http.DefaultTransport.(*http.Transport)

	return ClientConfig{
		MaxIdleConns:      defaultTransport.MaxIdleConns,
		IdleConnTimeout:   defaultTransport.IdleConnTimeout,
		ForceAttemptHTTP2: true,
	}
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

	transport.MaxIdleConns = cc.MaxIdleConns
	transport.MaxIdleConnsPerHost = cc.MaxIdleConnsPerHost
	transport.MaxConnsPerHost = cc.MaxConnsPerHost
	transport.IdleConnTimeout = cc.IdleConnTimeout
	transport.ForceAttemptHTTP2 = cc.ForceAttemptHTTP2

	// Setting the Proxy URL
	if cc.ProxyURL != "" {
		proxyURL, parseErr := url.ParseRequestURI(cc.ProxyURL)
		if parseErr != nil {
			return nil, parseErr
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	transport.DisableKeepAlives = cc.DisableKeepAlives

	if cc.HTTP2ReadIdleTimeout > 0 {
		transport2, transportErr := http2.ConfigureTransports(transport)
		if transportErr != nil {
			return nil, fmt.Errorf("failed to configure http2 transport: %w", transportErr)
		}
		transport2.ReadIdleTimeout = cc.HTTP2ReadIdleTimeout
		transport2.PingTimeout = cc.HTTP2PingTimeout
	}

	clientTransport := http.RoundTripper(transport)

	// Apply middlewares in reverse order so they execute in
	// forward order. The first middleware runs after authentication.
	if len(cc.Middlewares) > 0 && extensions == nil {
		return nil, errors.New("middlewares were configured but this component or its host does not support extensions")
	}
	for i := len(cc.Middlewares) - 1; i >= 0; i-- {
		var wrapper func(http.RoundTripper) (http.RoundTripper, error)
		wrapper, err = cc.Middlewares[i].GetHTTPClientRoundTripper(ctx, extensions)
		// If we failed to get the middleware
		if err != nil {
			return nil, err
		}
		clientTransport, err = wrapper(clientTransport)
		// If we failed to construct a wrapper
		if err != nil {
			return nil, err
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
