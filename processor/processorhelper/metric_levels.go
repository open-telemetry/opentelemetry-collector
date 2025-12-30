// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

func init() {
	component.RegisterMetricLevelConfigs(component.MetricLevelConfig{
		MeterName:      "go.opentelemetry.io/collector/processor/processorhelper",
		InstrumentName: "otelcol_processor_internal_duration",
		Level:          component.MetricLevel(configtelemetry.LevelDetailed),
	})
}
