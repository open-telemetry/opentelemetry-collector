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
)

const (
	// The value of "type" Attribute Key in configuration.
	typeStr = "memory_limiter"
)

// Factory is the factory for Attribute Key processor.
type Factory struct {
}

// Type gets the type of the config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor. Notice
// that the default configuration is expected to fail for this processor.
func (f *Factory) CreateDefaultConfig() configmodels.Processor {
	return generateDefaultConfig()
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateTraceProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (component.TraceProcessor, error) {
	return f.createProcessor(params.Logger, nextConsumer, nil, nil, cfg)
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (component.MetricsProcessor, error) {
	return f.createProcessor(params.Logger, nil, nextConsumer, nil, cfg)
}

// CreateLogsProcessor creates a metrics processor based on this config.
func (f *Factory) CreateLogProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.LogConsumer,
) (component.LogProcessor, error) {
	return f.createProcessor(params.Logger, nil, nil, nextConsumer, cfg)
}

var _ component.LogProcessorFactory = new(Factory)

type TripleTypeProcessor interface {
	consumer.TraceConsumer
	consumer.MetricsConsumer
	consumer.LogConsumer
	component.Processor
}

func (f *Factory) createProcessor(
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

func generateDefaultConfig() *Config {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}
