// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"go.opentelemetry.io/collector/component"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
)

type ObsMetrics = queuebatchtelemetry.ObsMetrics

// ExtractObsMetricsConfig unwraps repository-internal metrics from cfg.
func ExtractObsMetricsConfig(cfg component.Config, options []Option) (component.Config, []Option) {
	cfg, obsMetrics, ok := queuebatchtelemetry.ObsMetricsFromConfig(cfg)
	if !ok {
		return cfg, options
	}
	return cfg, append(options, withObsMetrics(obsMetrics))
}
