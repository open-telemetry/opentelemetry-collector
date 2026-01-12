// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//go:build linux || darwin

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func (na *AddrConfig) getListenConfig() (net.ListenConfig, error) {
	cfg := net.ListenConfig{}
	if na.ReusePort {
		cfg.Control = func(_, _ string, c syscall.RawConn) error {
			var controlErr error
			err := c.Control(func(fd uintptr) {
				controlErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			})
			if err != nil {
				return err
			}
			return controlErr
		}
	}

	return cfg, nil
}
