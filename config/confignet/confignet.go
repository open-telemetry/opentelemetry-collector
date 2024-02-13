// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
	"net"
	"time"
)

// DialerConfig contains options for connecting to an address.
type DialerConfig struct {
	// Timeout is the maximum amount of time a dial will wait for
	// a connect to complete. The default is no timeout.
	Timeout time.Duration `mapstructure:"timeout"`
}

// NetAddr represents a network endpoint address.
// Deprecated: [v0.95.0] Use AddrConfig instead.
type NetAddr = AddrConfig

// AddrConfig represents a network endpoint address.
type AddrConfig struct {
	// Endpoint configures the address for this network connection.
	// For TCP and UDP networks, the address has the form "host:port". The host must be a literal IP address,
	// or a host name that can be resolved to IP addresses. The port must be a literal port number or a service name.
	// If the host is a literal IPv6 address it must be enclosed in square brackets, as in "[2001:db8::1]:80" or
	// "[fe80::1%zone]:80". The zone specifies the scope of the literal IPv6 address as defined in RFC 4007.
	Endpoint string `mapstructure:"endpoint"`

	// Transport to use. Known protocols are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only), "udp", "udp4" (IPv4-only),
	// "udp6" (IPv6-only), "ip", "ip4" (IPv4-only), "ip6" (IPv6-only), "unix", "unixgram" and "unixpacket".
	Transport string `mapstructure:"transport"`

	// DialerConfig contains options for connecting to an address.
	DialerConfig DialerConfig `mapstructure:"dialer"`
}

// Dial equivalent with net.Dialer's DialContext for this address.
func (na *AddrConfig) Dial(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: na.DialerConfig.Timeout}
	return d.DialContext(ctx, na.Transport, na.Endpoint)
}

// Listen equivalent with net.ListenConfig's Listen for this address.
func (na *AddrConfig) Listen(ctx context.Context) (net.Listener, error) {
	lc := net.ListenConfig{}
	return lc.Listen(ctx, na.Transport, na.Endpoint)
}

// TCPAddr represents a TCP endpoint address.
// Deprecated: [v0.95.0] Use TCPAddrConfig instead.
type TCPAddr = TCPAddrConfig

// TCPAddrConfig represents a TCP endpoint address.
type TCPAddrConfig struct {
	// Endpoint configures the address for this network connection.
	// The address has the form "host:port". The host must be a literal IP address, or a host name that can be
	// resolved to IP addresses. The port must be a literal port number or a service name.
	// If the host is a literal IPv6 address it must be enclosed in square brackets, as in "[2001:db8::1]:80" or
	// "[fe80::1%zone]:80". The zone specifies the scope of the literal IPv6 address as defined in RFC 4007.
	Endpoint string `mapstructure:"endpoint"`

	// DialerConfig contains options for connecting to an address.
	DialerConfig DialerConfig `mapstructure:"dialer"`
}

// Dial equivalent with net.Dialer's DialContext for this address.
func (na *TCPAddrConfig) Dial(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: na.DialerConfig.Timeout}
	return d.DialContext(ctx, "tcp", na.Endpoint)
}

// Listen equivalent with net.ListenConfig's Listen for this address.
func (na *TCPAddrConfig) Listen(ctx context.Context) (net.Listener, error) {
	lc := net.ListenConfig{}
	return lc.Listen(ctx, "tcp", na.Endpoint)
}
