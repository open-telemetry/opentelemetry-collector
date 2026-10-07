// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
)

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
