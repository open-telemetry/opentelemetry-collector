// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//go:build windows

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"errors"
	"net"
)

func (sc *ServerConfig) getListenConfig() (net.ListenConfig, error) {
	if sc.ReusePort {
		return net.ListenConfig{}, errors.New("ReusePort is not supported on this platform")
	}

	return net.ListenConfig{}, nil
}
