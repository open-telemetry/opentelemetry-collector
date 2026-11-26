// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"go.opentelemetry.io/collector/config/configtls"
)

// dialTLSWithReloadedRootCAs returns a DialTLSContext function for transport
// that establishes each new TLS connection with a clone of the transport's
// TLSClientConfig whose RootCAs are set to the current pool returned by
// rootCAs. Server certificate verification is left to crypto/tls.
//
// It mirrors how net/http itself layers TLS over a plain connection, see
// persistConn.addTLS in
// https://github.com/golang/go/blob/go1.26.0/src/net/http/transport.go.
//
// net/http does not use DialTLSContext for the TLS session to the target
// through an HTTP proxy, so in that case the RootCAs of TLSClientConfig apply.
func dialTLSWithReloadedRootCAs(transport *http.Transport, rootCAs configtls.RootCAsFunc) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Like net/http, use the server name from the config if set, and the
		// host being dialed otherwise.
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		// Clone at dial time rather than once up front: net/http adds the
		// HTTP/2 ALPN protocol to TLSClientConfig when configuring HTTP/2.
		cfg := transport.TLSClientConfig.Clone()
		if cfg == nil {
			cfg = &tls.Config{}
		}
		cfg.RootCAs = rootCAs()
		if cfg.ServerName == "" {
			cfg.ServerName = host
		}

		rawConn, err := transport.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		if transport.TLSHandshakeTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, transport.TLSHandshakeTimeout)
			defer cancel()
		}
		tlsConn := tls.Client(rawConn, cfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		return tlsConn, nil
	}
}
