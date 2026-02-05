// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package promtest // import "go.opentelemetry.io/collector/service/internal/promtest"

import (
	"net"
	"strconv"
	"testing"

	"go.opentelemetry.io/contrib/otelconf"

	"go.opentelemetry.io/collector/internal/testutil"
)

func GetAvailableLocalIPv6AddressPrometheus(tb testing.TB) *otelconf.ExperimentalPrometheusMetricExporter {
	return addrToPrometheus(testutil.GetAvailableLocalIPv6Address(tb))
}

func GetAvailableLocalAddressPrometheus(tb testing.TB) *otelconf.ExperimentalPrometheusMetricExporter {
	return addrToPrometheus(testutil.GetAvailableLocalAddress(tb))
}

func addrToPrometheus(address string) *otelconf.ExperimentalPrometheusMetricExporter {
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
	return &otelconf.ExperimentalPrometheusMetricExporter{
		Host:                &host,
		Port:                &portInt,
		WithoutScopeInfo:    ptr(true),
		TranslationStrategy: ptr(otelconf.ExperimentalPrometheusTranslationStrategyUnderscoreEscapingWithoutSuffixes),
		WithResourceConstantLabels: &otelconf.IncludeExclude{
			Included: []string{},
		},
	}
}

func ptr[T any](v T) *T {
	return &v
}
