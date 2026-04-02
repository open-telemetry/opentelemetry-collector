// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"context"
	"net/url"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"
	"go.opentelemetry.io/collector/exporter/otlpexporter/internal/metadata"
	"go.opentelemetry.io/collector/exporter/xexporter"
)

// NewFactory creates a factory for OTLP exporter.
func NewFactory() exporter.Factory {
	return xexporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xexporter.WithDeprecatedTypeAlias(metadata.DeprecatedType),
		xexporter.WithTraces(createTraces, metadata.TracesStability),
		xexporter.WithMetrics(createMetrics, metadata.MetricsStability),
		xexporter.WithLogs(createLogs, metadata.LogsStability),
		xexporter.WithProfiles(createProfilesExporter, metadata.ProfilesStability),
	)
}

func createDefaultConfig() component.Config {
	clientCfg := configgrpc.NewDefaultClientConfig()
	// Default to gzip compression
	clientCfg.Compression = configcompression.TypeGzip
	// We almost read 0 bytes, so no need to tune ReadBufferSize.
	clientCfg.WriteBufferSize = 512 * 1024
	// For backward compatibility:
	clientCfg.Keepalive = configoptional.None[configgrpc.KeepaliveClientConfig]()

	return &Config{
		TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
		RetryConfig:   configretry.NewDefaultBackOffConfig(),
		QueueConfig:   configoptional.Some(exporterhelper.NewDefaultQueueConfig()),
		ClientConfig:  clientCfg,
	}
}

func endpointAttributes(cfg *Config) []attribute.KeyValue {
	endpointUrl, err := url.Parse(cfg.ClientConfig.Endpoint)
	if err != nil {
		// if the endpoint is not a URL, we treat it as the server address without extra information
		return []attribute.KeyValue{semconv.ServerAddressKey.String(cfg.ClientConfig.Endpoint)}
	}
	out := []attribute.KeyValue{
		semconv.ServerAddressKey.String(endpointUrl.Host),
		semconv.URLPath(endpointUrl.Path),
	}
	if endpointUrl.Port() != "" {
		out = append(out, semconv.ServerPortKey.String(endpointUrl.Port()))
	}
	return out
}

func createTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	oce := newExporter(cfg, set)
	oCfg := cfg.(*Config)

	return exporterhelper.NewTraces(ctx, set, cfg,
		oce.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
		exporterhelper.WithAttrs(endpointAttributes(oCfg)...),
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
		oce.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
		exporterhelper.WithAttrs(endpointAttributes(oCfg)...),
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
		oce.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
		exporterhelper.WithAttrs(endpointAttributes(oCfg)...),
	)
}

func createProfilesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (xexporter.Profiles, error) {
	oce := newExporter(cfg, set)
	oCfg := cfg.(*Config)
	return xexporterhelper.NewProfiles(ctx, set, cfg,
		oce.pushProfiles,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
		exporterhelper.WithAttrs(endpointAttributes(oCfg)...),
	)
}
