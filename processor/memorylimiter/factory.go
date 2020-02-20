// Copyright 2019, OpenTelemetry Authors
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
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

const (
	// The value of "type" Attribute Key in configuration.
	typeStr = "memory_limiter"
)

// Factory is the factory for Attribute Key processor.
type Factory struct {
}

// Type gets the type of the config created by this factory.
func (f *Factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor. Notice
// that the default configuration is expected to fail for this processor.
func (f *Factory) CreateDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (processor.TraceProcessor, error) {
	return f.createProcessor(logger, nextConsumer, nil, cfg)
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.MetricsProcessor, error) {
	return f.createProcessor(logger, nil, nextConsumer, cfg)
}

func (f *Factory) createProcessor(
	logger *zap.Logger,
	traceConsumer consumer.TraceConsumer,
	metricConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.DualTypeProcessor, error) {
	const mibBytes = 1024 * 1024
	pCfg := cfg.(*Config)
	return New(
		cfg.Name(),
		traceConsumer,
		metricConsumer,
		pCfg.CheckInterval,
		uint64(pCfg.MemoryLimitMiB)*mibBytes,
		uint64(pCfg.MemorySpikeLimitMiB)*mibBytes,
		uint64(pCfg.BallastSizeMiB)*mibBytes,
		logger,
	)
}
