// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"

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
	//TODO: Find a way to not have load the Pkl from a file? Load it from a string during an init()?
	// How would we create the string out of a Pkl file? Can we use Go Generate?
	cfg, err := LoadFromPath(context.Background(), `/Users/paulintodev/Documents/GitHub/opentelemetry-collector-paulin-f/processor/batchprocessor/config.pkl`)
	if err != nil {
		panic(err)
	}

	return cfg
}

func createTraces(
	_ context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	return newBatchTracesProcessor(set, nextConsumer, cfg.(*Config))
}

func createMetrics(
	_ context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	return newBatchMetricsProcessor(set, nextConsumer, cfg.(*Config))
}

func createLogs(
	_ context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	return newBatchLogsProcessor(set, nextConsumer, cfg.(*Config))
}
