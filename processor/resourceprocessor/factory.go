// Copyright The OpenTelemetry Authors
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

package resourceprocessor

import (
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

const (
	// The value of "type" key in configuration.
	typeStr = "resource"
)

// Factory is the factory for OpenCensus exporter.
type Factory struct {
}

// Type gets the type of the Option config created by this factory.
func (Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (Factory) CreateDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		ResourceType: "",
		Labels:       map[string]string{},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (Factory) CreateTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumerOld, cfg configmodels.Processor) (component.TraceProcessorOld, error) {
	oCfg := cfg.(*Config)
	return newResourceTraceProcessor(nextConsumer, oCfg), nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (Factory) CreateMetricsProcessor(logger *zap.Logger, nextConsumer consumer.MetricsConsumerOld, cfg configmodels.Processor) (component.MetricsProcessorOld, error) {
	oCfg := cfg.(*Config)
	return newResourceMetricProcessor(nextConsumer, oCfg), nil
}
