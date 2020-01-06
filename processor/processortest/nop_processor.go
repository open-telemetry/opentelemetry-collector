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

package processortest

import (
	"context"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type nopProcessor struct {
	nextTraceProcessor   consumer.TraceConsumer
	nextMetricsProcessor consumer.MetricsConsumer
}

var _ processor.TraceProcessor = (*nopProcessor)(nil)
var _ processor.MetricsProcessor = (*nopProcessor)(nil)

func (np *nopProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return np.nextTraceProcessor.ConsumeTraceData(ctx, td)
}

func (np *nopProcessor) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return np.nextMetricsProcessor.ConsumeMetricsData(ctx, md)
}

func (np *nopProcessor) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: false}
}

// Start is invoked during service startup.
func (np *nopProcessor) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (np *nopProcessor) Shutdown() error {
	return nil
}

// NewNopTraceProcessor creates an TraceProcessor that just pass the received data to the nextTraceProcessor.
func NewNopTraceProcessor(nextTraceProcessor consumer.TraceConsumer) consumer.TraceConsumer {
	return &nopProcessor{nextTraceProcessor: nextTraceProcessor}
}

// NewNopMetricsProcessor creates an MetricsProcessor that just pass the received data to the nextMetricsProcessor.
func NewNopMetricsProcessor(nextMetricsProcessor consumer.MetricsConsumer) consumer.MetricsConsumer {
	return &nopProcessor{nextMetricsProcessor: nextMetricsProcessor}
}

// NopProcessorFactory allows the creation of the no operation processor via
// config, so it can be used in tests that cannot create it directly.
type NopProcessorFactory struct{}

var _ processor.Factory = (*NopProcessorFactory)(nil)

// Type gets the type of the Processor created by this factory.
func (npf *NopProcessorFactory) Type() string {
	return "nop"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (npf *NopProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &configmodels.ProcessorSettings{
		TypeVal: npf.Type(),
		NameVal: npf.Type(),
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
// If the processor type does not support tracing or if the config is not valid
// error will be returned instead.
func (npf *NopProcessorFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (processor.TraceProcessor, error) {
	return &nopProcessor{nextTraceProcessor: nextConsumer}, nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
// If the processor type does not support metrics or if the config is not valid
// error will be returned instead.
func (npf *NopProcessorFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.MetricsProcessor, error) {
	return &nopProcessor{nextMetricsProcessor: nextConsumer}, nil
}
