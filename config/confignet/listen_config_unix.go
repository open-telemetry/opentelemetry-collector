// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux || darwin || freebsd || dragonfly || openbsd || netbsd

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func listenConfig(reusePort bool) (net.ListenConfig, error) {
	if !reusePort {
		return net.ListenConfig{}, nil
	}
	return net.ListenConfig{Control: setReusePort}, nil
}

func setReusePort(_, _ string, c syscall.RawConn) error {
	var opErr error
	if err := c.Control(func(fd uintptr) {
		opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	}); err != nil {
		return err
	}
	return opErr
}
