// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package promtest // import "go.opentelemetry.io/collector/service/internal/promtest"

import (
	"net"
	"strconv"
	"testing"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"

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
	if host == "::1" {
		host = "[::1]"
	}
	if err != nil {
		return nil
	}
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil
	}
	return &config.Prometheus{
		Host:              &host,
		Port:              &portInt,
		WithoutScopeInfo:  ptr(true),
		WithoutUnits:      ptr(true),
		WithoutTypeSuffix: ptr(true),
		WithResourceConstantLabels: &config.IncludeExclude{
			Included: []string{},
		},
	}
}

func ptr[T any](v T) *T {
	return &v
}
