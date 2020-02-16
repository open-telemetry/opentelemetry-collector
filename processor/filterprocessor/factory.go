// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterprocessor

import (
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

const (
	// The value of "type" key in configuration.
	typeStr = "filter"
)

// Factory is the factory for OpenCensus exporter.
type Factory struct {
}

var _ processor.Factory = Factory{}

// Type gets the type of the Option config created by this factory.
func (Factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (Factory) CreateDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (Factory) CreateTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumer, cfg configmodels.Processor) (processor.TraceProcessor, error) {
	oCfg := cfg.(*Config)
	return newFilterTraceProcessor(nextConsumer, oCfg)
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (Factory) CreateMetricsProcessor(logger *zap.Logger, nextConsumer consumer.MetricsConsumer, cfg configmodels.Processor) (processor.MetricsProcessor, error) {
	oCfg := cfg.(*Config)
	return newFilterMetricProcessor(nextConsumer, oCfg)
}
