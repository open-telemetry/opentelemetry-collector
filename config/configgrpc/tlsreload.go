// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc // import "go.opentelemetry.io/collector/config/configgrpc"

import (
	"context"
	"crypto/tls"
	"net"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/config/configtls"
)

// reloadingRootCAsCredentials are client TLS transport credentials that perform
// each handshake with a clone of tlsCfg whose RootCAs are set to the current
// pool returned by rootCAs. The handshake itself, including server certificate
// verification, is done by the standard gRPC TLS credentials.
type reloadingRootCAsCredentials struct {
	credentials.TransportCredentials

	tlsCfg  *tls.Config
	rootCAs configtls.RootCAsFunc
}

func newReloadingRootCAsCredentials(tlsCfg *tls.Config, rootCAs configtls.RootCAsFunc) credentials.TransportCredentials {
	return &reloadingRootCAsCredentials{
		TransportCredentials: credentials.NewTLS(tlsCfg),
		tlsCfg:               tlsCfg,
		rootCAs:              rootCAs,
	}
}

func (c *reloadingRootCAsCredentials) ClientHandshake(ctx context.Context, authority string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	cfg := c.tlsCfg.Clone()
	cfg.RootCAs = c.rootCAs()
	return credentials.NewTLS(cfg).ClientHandshake(ctx, authority, rawConn)
}

func (c *reloadingRootCAsCredentials) Clone() credentials.TransportCredentials {
	return &reloadingRootCAsCredentials{
		TransportCredentials: c.TransportCredentials.Clone(),
		tlsCfg:               c.tlsCfg.Clone(),
		rootCAs:              c.rootCAs,
	}
}
