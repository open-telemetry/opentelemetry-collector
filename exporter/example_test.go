// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporter_test

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// typeStr defines the unique type identifier for the exporter.
var typeStr = component.MustNewType("example")

// exampleConfig holds configuration settings for the exporter.
type exampleConfig struct {
	QueueSettings configoptional.Optional[exporterhelper.QueueBatchConfig]
	BackOffConfig configretry.BackOffConfig
}

// exampleExporter implements the OpenTelemetry exporter interface.
type exampleExporter struct {
	cancel context.CancelFunc
	config exampleConfig
	client *loggerClient
}

// loggerClient wraps a Zap logger to provide logging functionality.
type loggerClient struct {
	logger *zap.Logger
}

// Example demonstrates the usage of the exporter factory.
func Example() {
	// Instantiate the exporter factory and print its type.
	exampleExporter := NewFactory()
	fmt.Println(exampleExporter.Type())

	// Output:
	// example
}

// NewFactory creates a new exporter factory.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithMetrics(createExampleExporter, component.StabilityLevelAlpha),
	)
}

// createDefaultConfig returns the default configuration for the exporter.
func createDefaultConfig() component.Config {
	return &exampleConfig{}
}

// createExampleExporter initializes an instance of the example exporter.
func createExampleExporter(ctx context.Context, params exporter.Settings, baseCfg component.Config) (exporter.Metrics, error) {
	// Convert baseCfg to the correct type.
	cfg := baseCfg.(*exampleConfig)

	// Create a new exporter instance.
	xptr := newExampleExporter(ctx, cfg, params)

	// Wrap the exporter with the helper utilities.
	return exporterhelper.NewMetrics(
		ctx,
		params,
		cfg,
		xptr.consumeMetrics,
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithRetry(cfg.BackOffConfig),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithShutdown(xptr.shutdown),
	)
}

// newExampleExporter constructs a new instance of the example exporter.
func newExampleExporter(ctx context.Context, cfg *exampleConfig, params exporter.Settings) *exampleExporter {
	xptr := &exampleExporter{
		config: *cfg,
		client: &loggerClient{logger: params.Logger},
	}

	// Create a cancelable context.
	_, xptr.cancel = context.WithCancel(ctx)

	return xptr
}

// consumeMetrics processes incoming metric data and logs it.
func (xptr *exampleExporter) consumeMetrics(_ context.Context, md pmetric.Metrics) error {
	xptr.client.Push(md)
	return nil
}

// Shutdown properly stops the exporter and releases resources.
func (xptr *exampleExporter) shutdown(_ context.Context) error {
	xptr.cancel()
	return nil
}

// Push logs the received metric data.
func (client *loggerClient) Push(md pmetric.Metrics) {
	client.logger.Info("Exporting metrics", zap.Any("metrics", md))
}
