// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/internal"

import "go.opentelemetry.io/collector/component"

// Metrics receiver receives metrics.
// Its purpose is to translate data from any format to the collector's internal metrics format.
// MetricsReceiver feeds a consumer.Metrics with data.
//
// For example, it could be Prometheus data source which translates Prometheus metrics into pmetric.Metrics.
type Metrics interface {
	component.Component
}
