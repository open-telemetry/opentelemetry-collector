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

package nodebatcher

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
	typeStr = "batch"
)

// processorFactory is the factory for batch processor.
type processorFactory struct {
}

// Type gets the type of the config created by this factory.
func (f *processorFactory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *processorFactory) CreateDefaultConfig() models.Processor {
	removeAfterTicks := int(defaultRemoveAfterCycles)
	sendBatchSize := int(defaultSendBatchSize)
	tickTime := defaultTickTime
	timeout := defaultTimeout

	return &ConfigV2{
		ProcessorSettings: models.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		RemoveAfterTicks: &removeAfterTicks,
		SendBatchSize:    &sendBatchSize,
		NumTickers:       defaultNumTickers,
		TickTime:         &tickTime,
		Timeout:          &timeout,
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *processorFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	c models.Processor,
) (processor.TraceProcessor, error) {
	cfg := c.(*ConfigV2)

	var batchingOptions []Option
	if cfg.Timeout != nil {
		batchingOptions = append(batchingOptions, WithTimeout(*cfg.Timeout))
	}
	if cfg.NumTickers > 0 {
		batchingOptions = append(
			batchingOptions, WithNumTickers(cfg.NumTickers),
		)
	}
	if cfg.TickTime != nil {
		batchingOptions = append(
			batchingOptions, WithTickTime(*cfg.TickTime),
		)
	}
	if cfg.SendBatchSize != nil {
		batchingOptions = append(
			batchingOptions, WithSendBatchSize(*cfg.SendBatchSize),
		)
	}
	if cfg.RemoveAfterTicks != nil {
		batchingOptions = append(
			batchingOptions, WithRemoveAfterTicks(*cfg.RemoveAfterTicks),
		)
	}

	return NewBatcher(cfg.NameVal, logger, nextConsumer, batchingOptions...), nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *processorFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg models.Processor,
) (processor.MetricsProcessor, error) {
	return nil, models.ErrDataTypeIsNotSupported
}
