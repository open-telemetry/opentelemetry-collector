// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//go:build !windows

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func (sc *ServerConfig) getListenConfig() (net.ListenConfig, error) {
	cfg := net.ListenConfig{}
	if sc.ReusePort {
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
