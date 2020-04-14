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

package batchprocessor

import (
	"context"
	"time"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
)

const (
	// The value of "type" key in configuration.
	typeStr = "batch"

	defaultSendBatchSize = uint32(8192)
	defaultTimeout       = 200 * time.Millisecond
)

// Factory is the factory for batch processor.
type Factory struct {
}

// Type gets the type of the config created by this factory.
func (f *Factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *Factory) CreateDefaultConfig() configmodels.Processor {
	return generateDefaultConfig()
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateTraceProcessor(
	ctx context.Context,
	params component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	c configmodels.Processor,
) (component.TraceProcessor, error) {
	cfg := c.(*Config)
	return newBatchProcessor(params, nextConsumer, cfg), nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	ctx context.Context,
	params component.ProcessorCreateParams,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (component.MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

func generateDefaultConfig() *Config {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		SendBatchSize: defaultSendBatchSize,
		Timeout:       defaultTimeout,
	}
}
