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

package addattributesprocessor

import (
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/factories"
	"github.com/open-telemetry/opentelemetry-service/models"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

var _ = factories.RegisterProcessorFactory(&processorFactory{})

const (
	// The value of "type" key in configuration.
	typeStr = "attributes"
)

// processorFactory is the factory for Attributes processor.
type processorFactory struct {
}

// Type gets the type of the config created by this factory.
func (f *processorFactory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *processorFactory) CreateDefaultConfig() models.Processor {
	return &ConfigV2{
		ProcessorSettings: models.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Values: map[string]interface{}{},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *processorFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg models.Processor,
) (processor.TraceProcessor, error) {
	oCfg := cfg.(*ConfigV2)
	return NewTraceProcessor(nextConsumer, WithAttributes(oCfg.Values), WithOverwrite(oCfg.Overwrite))
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *processorFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg models.Processor,
) (processor.MetricsProcessor, error) {
	return nil, models.ErrDataTypeIsNotSupported
}
