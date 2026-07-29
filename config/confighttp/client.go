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
	"go.opentelemetry.io/collector/confmap"
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

	// MaxConnsPerHost limits the total number of connections per host, including connections in the dialing,
	// active, and idle states. Default is 0 (unlimited).
	MaxConnsPerHost int `mapstructure:"max_conns_per_host,omitempty"`

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

	// Keepalive configuration. Unmarshal folds this section into the deprecated
	// fields below, which remain the source of truth during their deprecation
	// window, and always resets it to None. A value visible to ToClient was
	// therefore set programmatically after unmarshaling (or the config was
	// never unmarshaled) and takes precedence over the deprecated fields.
	Keepalive configoptional.Optional[KeepaliveClientConfig] `mapstructure:"keepalive,omitempty"`

	// Deprecated: use Keepalive.IdleConnTimeout instead.
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout,omitempty"`
	// Deprecated: use Keepalive.MaxIdleConns instead.
	MaxIdleConns int `mapstructure:"max_idle_conns,omitempty"`
	// Deprecated: use Keepalive.MaxIdleConnsPerHost instead.
	MaxIdleConnsPerHost int `mapstructure:"max_idle_conns_per_host,omitempty"`
	// Deprecated: set 'keepalive::enabled' to false to disable keep-alives.
	DisableKeepAlives bool `mapstructure:"disable_keep_alives,omitempty"`

	// deprecationWarnings records use of deprecated fields observed while
	// unmarshaling; ToClient logs them, as no logger is available here.
	deprecationWarnings []string
}

// CookiesConfig defines the configuration of the HTTP client regarding cookies served by the server.
type CookiesConfig struct {
	_ struct{}
}

// KeepaliveClientConfig describes the keepalive configuration.
type KeepaliveClientConfig struct {
	_ struct{}

	// IdleConnTimeout is the maximum amount of time an iddle (keep-alive) connection will remain open before closing itself.
	// By default, it is set to 90 seconds.
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout"`

	// MaxIdleConns is used to set a limit to the maximum idle HTTP connections the client can keep open.
	// By default, it is set to 100. Zero means no limit.
	MaxIdleConns int `mapstructure:"max_idle_conns"`

	// MaxIdleConnsPerHost is used to set a limit to the maximum idle HTTP connections the host can keep open.
	// If zero, [net/http.DefaultMaxIdleConnsPerHost] is used.
	MaxIdleConnsPerHost int `mapstructure:"max_idle_conns_per_host,omitempty"`
}

// NewDefaultClientConfig returns ClientConfig type object with
// the default values of 'MaxIdleConns' and 'IdleConnTimeout', as well as [http.DefaultTransport] values.
// Other config options are not added as they are initialized with 'zero value' by GoLang as default.
// We encourage to use this function to create an object of ClientConfig.
func NewDefaultClientConfig() ClientConfig {
	// The default values are taken from the values of 'DefaultTransport' of 'http' package.
	defaultTransport := http.DefaultTransport.(*http.Transport)

	return ClientConfig{
		// The deprecated flat fields keep carrying the defaults so that
		// configurations and code which still use them behave as before.
		// Keepalive stays None; see its documentation.
		MaxIdleConns:      defaultTransport.MaxIdleConns,
		IdleConnTimeout:   defaultTransport.IdleConnTimeout,
		ForceAttemptHTTP2: true,
	}
}

var _ confmap.Unmarshaler = (*ClientConfig)(nil)

// Unmarshal implements confmap.Unmarshaler. It rejects configurations mixing the
// deprecated keepalive fields with the 'keepalive' section and records deprecation
// warnings for ToClient to log. Only keys present in the configuration count:
// values set programmatically (e.g. by a component's default config) are neither
// deprecated usage nor a conflict.
//
// The 'keepalive' section is folded into the deprecated fields, which stay the
// source of truth during their deprecation window, and Keepalive is always reset
// to None afterwards.
func (cc *ClientConfig) Unmarshal(conf *confmap.Conf) error {
	// Fold a Keepalive set programmatically (e.g. by a component's default
	// config) into the deprecated fields, so that the configuration decodes
	// over it below.
	if ka := cc.Keepalive.Get(); ka != nil {
		cc.IdleConnTimeout = ka.IdleConnTimeout
		cc.MaxIdleConns = ka.MaxIdleConns
		cc.MaxIdleConnsPerHost = ka.MaxIdleConnsPerHost
		cc.DisableKeepAlives = false
	}
	// Read before unmarshaling: decoding the Optional consumes the 'enabled' key.
	keepaliveDisabled := conf.Get("keepalive::enabled") == false

	// WithIgnoreUnused is needed because ClientConfig is commonly squash-embedded
	// into component configs, in which case conf also holds the parent's sibling fields.
	if err := conf.Unmarshal(cc, confmap.WithIgnoreUnused()); err != nil {
		return err
	}

	// A null 'keepalive' key carries no settings, but decodes as an enabled
	// section. Marshaling produces it for an unset Keepalive, so treat it as
	// unset to keep marshaled configurations loadable.
	keepaliveSet := conf.IsSet("keepalive") && conf.Get("keepalive") != nil

	// Values which match the field's zero value are no-ops in the legacy logic,
	// so they neither conflict with the 'keepalive' section nor deserve a warning.
	var deprecated []string
	if conf.IsSet("idle_conn_timeout") && cc.IdleConnTimeout != 0 {
		deprecated = append(deprecated, "'idle_conn_timeout' is deprecated; use 'keepalive::idle_conn_timeout' instead")
	}
	if conf.IsSet("max_idle_conns") && cc.MaxIdleConns != 0 {
		deprecated = append(deprecated, "'max_idle_conns' is deprecated; use 'keepalive::max_idle_conns' instead")
	}
	if conf.IsSet("max_idle_conns_per_host") && cc.MaxIdleConnsPerHost != 0 {
		deprecated = append(deprecated, "'max_idle_conns_per_host' is deprecated; use 'keepalive::max_idle_conns_per_host' instead")
	}
	if conf.IsSet("disable_keep_alives") && cc.DisableKeepAlives {
		deprecated = append(deprecated, "'disable_keep_alives' is deprecated; set 'keepalive::enabled' to false to disable keep-alives")
	}
	if keepaliveSet && len(deprecated) > 0 {
		return errors.New("confighttp.ClientConfig: cannot use deprecated keepalive fields (idle_conn_timeout, max_idle_conns, max_idle_conns_per_host, disable_keep_alives) alongside the 'keepalive' section; migrate to the 'keepalive' section")
	}
	cc.deprecationWarnings = deprecated

	// Fold the 'keepalive' section into the deprecated fields. Only keys
	// present in the configuration are copied; the deprecated fields supply
	// the values for the rest.
	if keepaliveSet {
		if ka := cc.Keepalive.Get(); ka != nil {
			if conf.IsSet("keepalive::idle_conn_timeout") {
				cc.IdleConnTimeout = ka.IdleConnTimeout
			}
			if conf.IsSet("keepalive::max_idle_conns") {
				cc.MaxIdleConns = ka.MaxIdleConns
			}
			if conf.IsSet("keepalive::max_idle_conns_per_host") {
				cc.MaxIdleConnsPerHost = ka.MaxIdleConnsPerHost
			}
		}
		// The section's presence determines keep-alives: enabled unless
		// 'keepalive::enabled' is false.
		cc.DisableKeepAlives = keepaliveDisabled
	}

	// Keepalive is always None after unmarshaling; see the field documentation.
	cc.Keepalive = configoptional.None[KeepaliveClientConfig]()
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
	} else {
		// Apply the deprecated flat fields exactly as the code before the
		// 'keepalive' section's introduction did; Unmarshal has already folded
		// the section into them.
		transport.DisableKeepAlives = cc.DisableKeepAlives
		transport.MaxIdleConns = cc.MaxIdleConns
		transport.MaxIdleConnsPerHost = cc.MaxIdleConnsPerHost
		transport.IdleConnTimeout = cc.IdleConnTimeout
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
