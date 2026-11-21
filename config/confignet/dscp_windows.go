// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"syscall"
)

const (
	ipprotoIP      = 0
	ipprotoIPv6    = 41
	ipTOS          = 3  // IP_TOS
	ipv6TrafficCls = 39 // IPV6_TCLASS
)

// DSCPDialControl returns a net.Dialer Control function that sets the DSCP value
// on the socket before connection. The DSCP value occupies the upper 6 bits of
// the IP ToS byte (IPv4) or Traffic Class byte (IPv6).
func DSCPDialControl(dscp int) func(string, string, syscall.RawConn) error {
	tosValue := dscp << 2 // DSCP occupies upper 6 bits; lower 2 bits are ECN
	return func(network, address string, c syscall.RawConn) error {
		var sysErr error
		err := c.Control(func(fd uintptr) {
			// On Windows, use Setsockopt via syscall handle.
			// IP_TOS may be silently ignored depending on Windows version and group policy.
			_ = syscall.SetsockoptInt(syscall.Handle(fd), ipprotoIP, ipTOS, tosValue)
			_ = syscall.SetsockoptInt(syscall.Handle(fd), ipprotoIPv6, ipv6TrafficCls, tosValue)
		})
		if err != nil {
			return err
		}
		return sysErr
	}
}
