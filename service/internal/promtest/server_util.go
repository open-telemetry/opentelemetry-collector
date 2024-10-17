// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package promtest // import "go.opentelemetry.io/collector/service/internal/promtest"

import (
	"fmt"
	"net"
	"strconv"
	"testing"

	"go.opentelemetry.io/contrib/config"

	"go.opentelemetry.io/collector/internal/testutil"
)

func GetAvailableLocalIPv6AddressPrometheus(t testing.TB) *config.Prometheus {
	p := addrToPrometheus(testutil.GetAvailableLocalIPv6Address(t))
	p.Host = ptr(fmt.Sprintf("[%s]", *p.Host))
	return p
}

func GetAvailableLocalAddressPrometheus(t testing.TB) *config.Prometheus {
	return addrToPrometheus(testutil.GetAvailableLocalAddress(t))
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
		Host:              &host,
		Port:              &portInt,
		WithoutScopeInfo:  ptr(true),
		WithoutUnits:      ptr(true),
		WithoutTypeSuffix: ptr(true),
		WithResourceConstantLabels: &config.IncludeExclude{
			Excluded: []string{},
		},
	}
}

func ptr[T any](v T) *T {
	return &v
}
