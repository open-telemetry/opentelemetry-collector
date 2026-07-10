// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// TransportType represents a type of network transport protocol
type TransportType string

const (
	TransportTypeTCP        TransportType = "tcp"
	TransportTypeTCP4       TransportType = "tcp4"
	TransportTypeTCP6       TransportType = "tcp6"
	TransportTypeUDP        TransportType = "udp"
	TransportTypeUDP4       TransportType = "udp4"
	TransportTypeUDP6       TransportType = "udp6"
	TransportTypeIP         TransportType = "ip"
	TransportTypeIP4        TransportType = "ip4"
	TransportTypeIP6        TransportType = "ip6"
	TransportTypeUnix       TransportType = "unix"
	TransportTypeUnixgram   TransportType = "unixgram"
	TransportTypeUnixPacket TransportType = "unixpacket"
	TransportTypeNpipe      TransportType = "npipe"
	TransportTypeVsock      TransportType = "vsock"
	transportTypeEmpty      TransportType = ""
)

// UnmarshalText unmarshalls text to a TransportType.
// Valid values are "tcp", "tcp4", "tcp6", "udp", "udp4",
// "udp6", "ip", "ip4", "ip6", "unix", "unixgram", "unixpacket", "npipe" and "vsock"
func (tt *TransportType) UnmarshalText(in []byte) error {
	typ := TransportType(in)
	switch typ {
	case TransportTypeTCP,
		TransportTypeTCP4,
		TransportTypeTCP6,
		TransportTypeUDP,
		TransportTypeUDP4,
		TransportTypeUDP6,
		TransportTypeIP,
		TransportTypeIP4,
		TransportTypeIP6,
		TransportTypeUnix,
		TransportTypeUnixgram,
		TransportTypeUnixPacket,
		TransportTypeNpipe,
		TransportTypeVsock,
		transportTypeEmpty:
		*tt = typ
		return nil
	default:
		return fmt.Errorf("unsupported transport type %q", typ)
	}
}

// DialerConfig contains options for connecting to an address.
type DialerConfig struct {
	// Timeout is the maximum amount of time a dial will wait for
	// a connect to complete. The default is no timeout.
	Timeout time.Duration `mapstructure:"timeout,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultDialerConfig creates a new DialerConfig with any default values set
func NewDefaultDialerConfig() DialerConfig {
	return DialerConfig{}
}

// AddrConfig represents a network endpoint address.
type AddrConfig struct {
	// Endpoint configures the address for this network connection.
	// For TCP and UDP networks, the address has the form "host:port". The host must be a literal IP address,
	// or a host name that can be resolved to IP addresses. The port must be a literal port number or a service name.
	// If the host is a literal IPv6 address it must be enclosed in square brackets, as in "[2001:db8::1]:80" or
	// "[fe80::1%zone]:80". The zone specifies the scope of the literal IPv6 address as defined in RFC 4007.
	Endpoint string `mapstructure:"endpoint,omitempty"`

	// Transport to use. Allowed protocols are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only), "udp", "udp4" (IPv4-only),
	// "udp6" (IPv6-only), "ip", "ip4" (IPv4-only), "ip6" (IPv6-only), "unix", "unixgram", "unixpacket",
	// "npipe" (Windows named pipes, Windows-only) and "vsock" (VM sockets, Linux-only).
	// For vsock, the endpoint must be in the form "cid:port" where cid is the VM context ID and port
	// is a 32-bit port number.
	Transport TransportType `mapstructure:"transport,omitempty"`

	// DialerConfig contains options for connecting to an address.
	DialerConfig DialerConfig `mapstructure:"dialer,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultAddrConfig creates a new AddrConfig with any default values set
func NewDefaultAddrConfig() AddrConfig {
	return AddrConfig{
		DialerConfig: NewDefaultDialerConfig(),
	}
}

// Dial equivalent with net.Dialer's DialContext for this address.
func (na *AddrConfig) Dial(ctx context.Context) (net.Conn, error) {
	switch na.Transport {
	case TransportTypeNpipe:
		return dialNpipe(ctx, na.Endpoint, na.DialerConfig.Timeout)
	case TransportTypeVsock:
		return dialVsock(ctx, na.Endpoint, na.DialerConfig.Timeout)
	}
	d := net.Dialer{Timeout: na.DialerConfig.Timeout}
	return d.DialContext(ctx, string(na.Transport), na.Endpoint)
}

// Listen equivalent with net.ListenConfig's Listen for this address.
func (na *AddrConfig) Listen(ctx context.Context) (net.Listener, error) {
	switch na.Transport {
	case TransportTypeNpipe:
		return listenNpipe(na.Endpoint)
	case TransportTypeVsock:
		return listenVsock(na.Endpoint)
	}
	lc := net.ListenConfig{}
	return lc.Listen(ctx, string(na.Transport), na.Endpoint)
}

func (na *AddrConfig) Validate() error {
	switch na.Transport {
	case TransportTypeTCP,
		TransportTypeTCP4,
		TransportTypeTCP6,
		TransportTypeUDP,
		TransportTypeUDP4,
		TransportTypeUDP6,
		TransportTypeIP,
		TransportTypeIP4,
		TransportTypeIP6,
		TransportTypeUnix,
		TransportTypeUnixgram,
		TransportTypeUnixPacket:
		return nil
	case TransportTypeNpipe:
		return validateNpipePath(na.Endpoint)
	case TransportTypeVsock:
		return validateVsockEndpoint(na.Endpoint)
	default:
		return fmt.Errorf("invalid transport type %q", na.Transport)
	}
}

// validateNpipePath validates a Windows named pipe path.
// Named pipe paths must follow the format: \\<server>\pipe\<name>
// See: https://learn.microsoft.com/en-us/windows/win32/ipc/pipe-names
func validateNpipePath(endpoint string) error {
	const maxLen = 256
	if len(endpoint) > maxLen {
		return fmt.Errorf("named pipe path %q exceeds maximum length of %d characters", endpoint, maxLen)
	}
	if !strings.HasPrefix(endpoint, `\\`) {
		return fmt.Errorf(`named pipe path must start with "\\": %q`, endpoint)
	}
	// After \\, find the \pipe\ component (case-insensitive per Windows rules)
	rest := strings.ToLower(endpoint[2:])
	pipeIdx := strings.Index(rest, `\pipe\`)
	if pipeIdx < 0 {
		return fmt.Errorf(`named pipe path must contain "\pipe\": %q`, endpoint)
	}
	if pipeIdx == 0 {
		return fmt.Errorf("named pipe path must have a non-empty server name: %q", endpoint)
	}
	pipeName := endpoint[2+pipeIdx+len(`\pipe\`):]
	if pipeName == "" {
		return fmt.Errorf(`named pipe path must have a non-empty pipe name after "\pipe\": %q`, endpoint)
	}
	if strings.ContainsRune(pipeName, '\\') {
		return fmt.Errorf("named pipe name must not contain backslashes: %q", endpoint)
	}
	return nil
}

func validateVsockEndpoint(endpoint string) error {
	_, _, err := parseVsockEndpoint(endpoint)
	return err
}

func parseVsockEndpoint(endpoint string) (uint32, uint32, error) {
	idx := strings.LastIndex(endpoint, ":")
	if idx <= 0 || idx == len(endpoint)-1 {
		return 0, 0, fmt.Errorf("vsock endpoint must be in the form \"cid:port\": %q", endpoint)
	}
	cidStr := endpoint[:idx]
	portStr := endpoint[idx+1:]
	cid, err := strconv.ParseUint(cidStr, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("vsock endpoint has invalid context ID %q: %w", cidStr, err)
	}
	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("vsock endpoint has invalid port %q: %w", portStr, err)
	}
	return uint32(cid), uint32(port), nil
}

// TCPAddrConfig represents a TCP endpoint address.
type TCPAddrConfig struct {
	// Endpoint configures the address for this network connection.
	// The address has the form "host:port". The host must be a literal IP address, or a host name that can be
	// resolved to IP addresses. The port must be a literal port number or a service name.
	// If the host is a literal IPv6 address it must be enclosed in square brackets, as in "[2001:db8::1]:80" or
	// "[fe80::1%zone]:80". The zone specifies the scope of the literal IPv6 address as defined in RFC 4007.
	Endpoint string `mapstructure:"endpoint,omitempty"`

	// DialerConfig contains options for connecting to an address.
	DialerConfig DialerConfig `mapstructure:"dialer,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultTCPAddrConfig creates a new TCPAddrConfig with any default values set
func NewDefaultTCPAddrConfig() TCPAddrConfig {
	return TCPAddrConfig{
		DialerConfig: NewDefaultDialerConfig(),
	}
}

// Dial equivalent with net.Dialer's DialContext for this address.
func (na *TCPAddrConfig) Dial(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: na.DialerConfig.Timeout}
	return d.DialContext(ctx, string(TransportTypeTCP), na.Endpoint)
}

// Listen equivalent with net.ListenConfig's Listen for this address.
func (na *TCPAddrConfig) Listen(ctx context.Context) (net.Listener, error) {
	lc := net.ListenConfig{}
	return lc.Listen(ctx, string(TransportTypeTCP), na.Endpoint)
}
