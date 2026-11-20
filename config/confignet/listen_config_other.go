// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !darwin && !freebsd && !dragonfly && !openbsd && !netbsd

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"errors"
	"net"
)

func listenConfig(reusePort bool) (net.ListenConfig, error) {
	if reusePort {
		return net.ListenConfig{}, errors.New("reuse_port is not supported on this platform")
	}
	return net.ListenConfig{}, nil
}
