// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memorylimiter

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// The value of "type" Attribute Key in configuration.
	typeStr = "memory_limiter"
)

// NewFactory returns a new factory for the Memory Limiter processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTraceProcessor),
		processorhelper.WithMetrics(createMetricsProcessor),
		processorhelper.WithLogs(createLogProcessor))
}

// CreateDefaultConfig creates the default configuration for processor. Notice
// that the default configuration is expected to fail for this processor.
func createDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

func createTraceProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TraceConsumer,
) (component.TraceProcessor, error) {
	return createProcessor(params.Logger, nextConsumer, nil, nil, cfg)
}

func createMetricsProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.MetricsConsumer,
) (component.MetricsProcessor, error) {
	return createProcessor(params.Logger, nil, nextConsumer, nil, cfg)
}

func createLogProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.LogConsumer,
) (component.LogProcessor, error) {
	return createProcessor(params.Logger, nil, nil, nextConsumer, cfg)
}

type TripleTypeProcessor interface {
	consumer.TraceConsumer
	consumer.MetricsConsumer
	consumer.LogConsumer
	component.Processor
}

func createProcessor(
	logger *zap.Logger,
	traceConsumer consumer.TraceConsumer,
	metricConsumer consumer.MetricsConsumer,
	logConsumer consumer.LogConsumer,
	cfg configmodels.Processor,
) (TripleTypeProcessor, error) {
	pCfg := cfg.(*Config)
	return newMemoryLimiter(
		logger,
		traceConsumer,
		metricConsumer,
		logConsumer,
		pCfg,
	)
}
