// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//go:build !(linux || darwin)

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"errors"
	"net"
)

func (na *AddrConfig) getListenConfig() (net.ListenConfig, error) {
	if na.ReusePort {
		return net.ListenConfig{}, errors.New("ReusePort is not supported on this platform")
	}

	return net.ListenConfig{}, nil
}
