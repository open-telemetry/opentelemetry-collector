// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

func init() {
	component.RegisterMetricLevelConfigs(
		component.MetricLevelConfig{
			MeterName: "go.opentelemetry.io/collector/processor/batchprocessor",
			Level:     component.MetricLevel(configtelemetry.LevelNormal),
		},
		component.MetricLevelConfig{
			MeterName:      "go.opentelemetry.io/collector/processor/batchprocessor",
			InstrumentName: "otelcol_processor_batch_batch_send_size_bytes",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
	)
}
