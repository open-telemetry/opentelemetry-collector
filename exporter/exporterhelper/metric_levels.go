// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

func init() {
	component.RegisterMetricLevelConfigs(component.MetricLevelConfig{
		MeterName:      "go.opentelemetry.io/collector/exporter/exporterhelper",
		InstrumentName: "otelcol_exporter_queue_batch_send_size_bytes",
		Level:          component.MetricLevel(configtelemetry.LevelDetailed),
	})
}
