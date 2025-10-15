// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
)

const (
	defaultSendBatchSize = uint32(8192)
	defaultTimeout       = 200 * time.Millisecond

	// defaultMetadataCardinalityLimit should be set to the number
	// of metadata configurations the user expects to submit to
	// the collector.
	defaultMetadataCardinalityLimit = 1000
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
	return &Config{
		SendBatchSize:            defaultSendBatchSize,
		Timeout:                  defaultTimeout,
		MetadataCardinalityLimit: defaultMetadataCardinalityLimit,
	}
}

func createTraces(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	batchCfg := cfg.(*Config)
	if useExporterHelper.IsEnabled() {
		return newTracesProcessorWithExporterHelper(ctx, set, nextConsumer, batchCfg)
	}
	return newTracesBatchProcessor(ctx, set, nextConsumer, batchCfg)
}

func createMetrics(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	batchCfg := cfg.(*Config)
	if useExporterHelper.IsEnabled() {
		return newMetricsProcessorWithExporterHelper(ctx, set, nextConsumer, batchCfg)
	}
	return newMetricsBatchProcessor(ctx, set, nextConsumer, batchCfg)
}

func createLogs(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	batchCfg := cfg.(*Config)
	if useExporterHelper.IsEnabled() {
		return newLogsProcessorWithExporterHelper(ctx, set, nextConsumer, batchCfg)
	}
	return newLogsBatchProcessor(ctx, set, nextConsumer, batchCfg)
}
