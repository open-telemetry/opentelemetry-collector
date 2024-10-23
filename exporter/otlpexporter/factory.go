// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/exporterhelperprofiles"
	"go.opentelemetry.io/collector/exporter/exporterprofiles"
	"go.opentelemetry.io/collector/exporter/otlpexporter/internal/metadata"
)

// NewFactory creates a factory for OTLP exporter.
func NewFactory() exporter.Factory {
	return exporterprofiles.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporterprofiles.WithTraces(createTraces, metadata.TracesStability),
		exporterprofiles.WithMetrics(createMetrics, metadata.MetricsStability),
		exporterprofiles.WithLogs(createLogs, metadata.LogsStability),
		exporterprofiles.WithProfiles(createProfilesExporter, metadata.ProfilesStability),
	)
}

func createDefaultConfig() component.Config {
	batcherCfg := exporterbatcher.NewDefaultConfig()
	batcherCfg.Enabled = false

	return &Config{
		TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
		RetryConfig:   configretry.NewDefaultBackOffConfig(),
		QueueConfig:   exporterhelper.NewDefaultQueueConfig(),
		BatcherConfig: batcherCfg,
		ClientConfig: configgrpc.ClientConfig{
			Headers: map[string]configopaque.String{},
			// Default to gzip compression
			Compression: configcompression.TypeGzip,
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
}

func createTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	oce := newExporter(cfg, set)
	oCfg := cfg.(*Config)
	return exporterhelper.NewTraces(ctx, set, cfg,
		oce.pushTracesWithStatus,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithBatcher(oCfg.BatcherConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
	)
}

func createMetrics(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	oce := newExporter(cfg, set)
	oCfg := cfg.(*Config)
	return exporterhelper.NewMetrics(ctx, set, cfg,
		oce.pushMetricsWithStatus,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithBatcher(oCfg.BatcherConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
	)
}

func createLogs(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	oce := newExporter(cfg, set)
	oCfg := cfg.(*Config)
	return exporterhelper.NewLogs(ctx, set, cfg,
		oce.pushLogsWithStatus,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithBatcher(oCfg.BatcherConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
	)
}

func createProfilesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporterprofiles.Profiles, error) {
	oce := newExporter(cfg, set)
	oCfg := cfg.(*Config)
	return exporterhelperprofiles.NewProfilesExporter(ctx, set, cfg,
		oce.pushProfilesWithStatus,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithBatcher(oCfg.BatcherConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
	)
}
