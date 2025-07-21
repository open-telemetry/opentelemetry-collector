// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper_test

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// typeStr defines the unique type identifier for the processor.
var typeStr = component.MustNewType("example")

// exampleConfig holds configuration settings for the processor.
type exampleConfig struct{}

// exampleProcessor implements the OpenTelemetry processor interface.
type exampleProcessor struct {
	cancel context.CancelFunc
	config exampleConfig
}

// Example demonstrates the usage of the processor factory.
func Example() {
	// Instantiate the processor factory and print its type.
	exampleProcessor := NewFactory()
	fmt.Println(exampleProcessor.Type())

	// Output:
	// example
}

// NewFactory creates a new processor factory.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithMetrics(createExampleProcessor, component.StabilityLevelAlpha),
	)
}

// createDefaultConfig returns the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &exampleConfig{}
}

// createExampleProcessor initializes an instance of the example processor.
func createExampleProcessor(ctx context.Context, params processor.Settings, baseCfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	// Convert baseCfg to the correct type.
	cfg := baseCfg.(*exampleConfig)

	// Create a new processor instance.
	pcsr := newExampleProcessor(ctx, cfg)

	// Wrap the processor with the helper utilities.
	return processorhelper.NewMetrics(
		ctx,
		params,
		cfg,
		next,
		pcsr.consumeMetrics,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
		processorhelper.WithShutdown(pcsr.shutdown),
	)
}

// newExampleProcessor constructs a new instance of the example processor.
func newExampleProcessor(ctx context.Context, cfg *exampleConfig) *exampleProcessor {
	pcsr := &exampleProcessor{
		config: *cfg,
	}

	// Create a cancelable context.
	_, pcsr.cancel = context.WithCancel(ctx)

	return pcsr
}

// ConsumeMetrics modify metrics adding one attribute to resource.
func (pcsr *exampleProcessor) consumeMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		rm.At(i).Resource().Attributes().PutStr("processed_by", "exampleProcessor")
	}

	return md, nil
}

// Shutdown properly stops the processor and releases resources.
func (pcsr *exampleProcessor) shutdown(_ context.Context) error {
	pcsr.cancel()
	return nil
}
