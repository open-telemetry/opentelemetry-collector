// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package promtest // import "go.opentelemetry.io/collector/service/internal/promtest"

import (
	"net"
	"strconv"
	"testing"

	"go.opentelemetry.io/contrib/config"

	"go.opentelemetry.io/collector/internal/testutil"
)

func GetAvailableLocalIPv6AddressPrometheus(tb testing.TB) *config.Prometheus {
	return addrToPrometheus(testutil.GetAvailableLocalIPv6Address(tb))
}

func GetAvailableLocalAddressPrometheus(tb testing.TB) *config.Prometheus {
	return addrToPrometheus(testutil.GetAvailableLocalAddress(tb))
}

func addrToPrometheus(address string) *config.Prometheus {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil
	}
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil
	}
	return &config.Prometheus{
		Host: &host,
		Port: &portInt,
	}
}
