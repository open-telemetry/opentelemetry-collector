//go:generate mdatagen metadata.yaml

package pipelineprocessor // import "go.opentelemetry.io/collector/processor/pipelineprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/pipelineprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

const (
	// defaultTimeout is the default timeout for the processor.
	// 0 means no timeout is applied, allowing the context to pass through.
	defaultTimeout = 0
)

// NewFactory returns a new factory for the Pipeline processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTraces, metadata.TracesStability),
		processor.WithMetrics(createMetrics, metadata.MetricsStability),
		processor.WithLogs(createLogs, metadata.LogsStability))
}

func createDefaultConfig() component.Config {
	queueConfig := exporterhelper.NewDefaultQueueConfig()
	queueConfig.Enabled = false

	retryConfig := configretry.NewDefaultBackOffConfig()
	retryConfig.Enabled = false

	return &Config{
		TimeoutConfig: exporterhelper.TimeoutConfig{
			Timeout: defaultTimeout,
		},
		QueueConfig: queueConfig,
		RetryConfig: retryConfig,
	}
}

func createTraces(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	return newTracesProcessor(set, nextConsumer, cfg.(*Config))
}

func createMetrics(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	return newMetricsProcessor(set, nextConsumer, cfg.(*Config))
}

func createLogs(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	return newLogsProcessor(set, nextConsumer, cfg.(*Config))
}

// NewFactoryWithProfiles returns a new factory for the Pipeline processor with experimental profiles support.
func NewFactoryWithProfiles() xprocessor.Factory {
	return xprocessor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xprocessor.WithTraces(createTraces, metadata.TracesStability),
		xprocessor.WithMetrics(createMetrics, metadata.MetricsStability),
		xprocessor.WithLogs(createLogs, metadata.LogsStability),
		xprocessor.WithProfiles(createProfiles, metadata.ProfilesStability))
}

func createProfiles(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xprocessor.Profiles, error) {
	return newProfilesProcessor(set, nextConsumer, cfg.(*Config))
}
