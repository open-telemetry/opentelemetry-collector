// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/obsmetrics"

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
//
// Implementations must be updated when exporterhelper adds a metric.
type ObsMetrics = obsmetrics.ObsMetrics
