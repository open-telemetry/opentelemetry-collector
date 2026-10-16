// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"syscall"
)

// DSCPDialControl returns a net.Dialer Control function that sets the DSCP value
// on the socket before connection. The DSCP value occupies the upper 6 bits of
// the IP ToS byte (IPv4) or Traffic Class byte (IPv6).
func DSCPDialControl(dscp int) func(string, string, syscall.RawConn) error {
	tosValue := dscp << 2 // DSCP occupies upper 6 bits; lower 2 bits are ECN
	return func(network, address string, c syscall.RawConn) error {
		var sysErr error
		err := c.Control(func(fd uintptr) {
			// Try both IPv4 and IPv6; the irrelevant one will return an error
			// which we ignore since we don't know the address family at this point.
			_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TOS, tosValue)
			_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_TCLASS, tosValue)
		})
		if err != nil {
			return err
		}
		return sysErr
	}
}
