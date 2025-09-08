// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
)

// NewFactory returns a new factory for the Batch processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTraces, metadata.TracesStability),
		processor.WithMetrics(createMetrics, metadata.MetricsStability),
		processor.WithLogs(createLogs, metadata.LogsStability))
}

func createDefaultConfig() component.Config {
	cfg := &Config{}

	// Unmarshal empty JSON to get default values.
	// Errors shouldn't happen, so we can ignore them.
	// TODO: Find a cleaner, more mapstructure-friendly way to do this.
	// For example: https://github.com/omissis/go-jsonschema/pull/283
	_ = json.Unmarshal([]byte("{}"), cfg)

	return cfg
}

func createTraces(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	return newTracesBatchProcessor(set, nextConsumer, cfg.(*Config))
}

func createMetrics(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	return newMetricsBatchProcessor(set, nextConsumer, cfg.(*Config))
}

func createLogs(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	return newLogsBatchProcessor(set, nextConsumer, cfg.(*Config))
}
