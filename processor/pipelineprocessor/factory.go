//go:generate mdatagen metadata.yaml

package pipelineprocessor // import "go.opentelemetry.io/collector/processor/pipelineprocessor"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/pipelineprocessor/internal/metadata"
)

const (
	// defaultTimeout is the default timeout for the processor.
	defaultTimeout = 5 * time.Second
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
	return &Config{
		TimeoutConfig: exporterhelper.TimeoutConfig{
			Timeout: defaultTimeout,
		},
		QueueConfig: exporterhelper.NewDefaultQueueConfig(),
		RetryConfig: configretry.NewDefaultBackOffConfig(),
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
