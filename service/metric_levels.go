// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

func init() {
	component.RegisterMetricLevelConfigs(
		// otelhttp and otelgrpc instrumentation.
		component.MetricLevelConfig{
			MeterName: "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
			Level:     component.MetricLevel(configtelemetry.LevelDetailed),
		},
		component.MetricLevelConfig{
			MeterName: "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
			Level:     component.MetricLevel(configtelemetry.LevelDetailed),
		},
		// otel-arrow receiver metrics.
		component.MetricLevelConfig{
			MeterName:      "otel-arrow/pkg/otel/arrow_record",
			InstrumentName: "arrow_batch_records",
			Level:          component.MetricLevel(configtelemetry.LevelNormal),
		},
		component.MetricLevelConfig{
			MeterName:      "otel-arrow/pkg/otel/arrow_record",
			InstrumentName: "arrow_schema_resets",
			Level:          component.MetricLevel(configtelemetry.LevelNormal),
		},
		component.MetricLevelConfig{
			MeterName:      "otel-arrow/pkg/otel/arrow_record",
			InstrumentName: "arrow_memory_inuse",
			Level:          component.MetricLevel(configtelemetry.LevelNormal),
		},
		// Contrib netstats metrics.
		component.MetricLevelConfig{
			MeterName:      "github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow/netstats",
			InstrumentName: "otelcol_*_compressed_size",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
		component.MetricLevelConfig{
			MeterName:      "github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow/netstats",
			InstrumentName: "otelcol_exporter_recv",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
		component.MetricLevelConfig{
			MeterName:      "github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow/netstats",
			InstrumentName: "otelcol_exporter_recv_wire",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
		component.MetricLevelConfig{
			MeterName:      "github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow/netstats",
			InstrumentName: "otelcol_receiver_sent",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
		component.MetricLevelConfig{
			MeterName:      "github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow/netstats",
			InstrumentName: "otelcol_receiver_sent_wire",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
		// Internal service graph metrics.
		component.MetricLevelConfig{
			MeterName:      "go.opentelemetry.io/collector/service",
			InstrumentName: "otelcol.*.consumed.size",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
		component.MetricLevelConfig{
			MeterName:      "go.opentelemetry.io/collector/service",
			InstrumentName: "otelcol.*.produced.size",
			Level:          component.MetricLevel(configtelemetry.LevelDetailed),
		},
	)
}
