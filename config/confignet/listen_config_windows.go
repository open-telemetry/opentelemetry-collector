// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//go:build windows

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"errors"
	"net"
)

func (sc *AddrConfig) getListenConfig() (net.ListenConfig, error) {
	if sc.ReusePort {
		return net.ListenConfig{}, errors.New("ReusePort is not supported on this platform")
	}

	return net.ListenConfig{}, nil
}
