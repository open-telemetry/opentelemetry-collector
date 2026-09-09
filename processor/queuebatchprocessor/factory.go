// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

// NewFactory returns a new factory for the queue/batch processor.
func NewFactory() processor.Factory {
	return xprocessor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xprocessor.WithTraces(createTraces, metadata.TracesStability),
		xprocessor.WithMetrics(createMetrics, metadata.MetricsStability),
		xprocessor.WithLogs(createLogs, metadata.LogsStability),
		xprocessor.WithProfiles(createProfiles, metadata.ProfilesStability),
	)
}

// createDefaultConfig returns the default configuration, modified
// from exporterhelper.NewDefaultQueueConfig() to enable backpressure,
// limit concurrency to 1, w/ queue-size 10.
func createDefaultConfig() component.Config {
	cfg := exporterhelper.NewDefaultQueueConfig()

	// These are all different from exporterhelper.
	cfg.BlockOnOverflow = true
	cfg.NumConsumers = 1
	cfg.QueueSize = 10

	// Enable batching with default settings.
	_ = cfg.Batch.GetOrInsertDefault()
	return &cfg
}

func createTraces(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Traces) (processor.Traces, error) {
	return newTracesProcessor(ctx, set, cfg.(*Config), next)
}

func createMetrics(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	return newMetricsProcessor(ctx, set, cfg.(*Config), next)
}

func createLogs(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
	return newLogsProcessor(ctx, set, cfg.(*Config), next)
}

func createProfiles(ctx context.Context, set processor.Settings, cfg component.Config, next xconsumer.Profiles) (xprocessor.Profiles, error) {
	return newProfilesProcessor(ctx, set, cfg.(*Config), next)
}
