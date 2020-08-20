// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
)

type nopProcessor struct {
	nextTraceProcessor   consumer.TraceConsumerOld
	nextMetricsProcessor consumer.MetricsConsumerOld
}

func (np *nopProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return np.nextTraceProcessor.ConsumeTraceData(ctx, td)
}

func (np *nopProcessor) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return np.nextMetricsProcessor.ConsumeMetricsData(ctx, md)
}

func (np *nopProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Start is invoked during service startup.
func (np *nopProcessor) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (np *nopProcessor) Shutdown(context.Context) error {
	return nil
}

// NewNopTraceProcessor creates an TraceProcessor that just pass the received data to the nextTraceProcessor.
func NewNopTraceProcessor(nextTraceProcessor consumer.TraceConsumerOld) consumer.TraceConsumerOld {
	return &nopProcessor{nextTraceProcessor: nextTraceProcessor}
}

// NewNopMetricsProcessor creates an MetricsProcessor that just pass the received data to the nextMetricsProcessor.
func NewNopMetricsProcessor(nextMetricsProcessor consumer.MetricsConsumerOld) consumer.MetricsConsumerOld {
	return &nopProcessor{nextMetricsProcessor: nextMetricsProcessor}
}

// NopProcessorFactory allows the creation of the no operation processor via
// config, so it can be used in tests that cannot create it directly.
type NopProcessorFactory struct{}

// Type gets the type of the Processor created by this factory.
func (npf *NopProcessorFactory) Type() configmodels.Type {
	return "nop"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (npf *NopProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &configmodels.ProcessorSettings{
		TypeVal: npf.Type(),
		NameVal: string(npf.Type()),
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
// If the processor type does not support tracing or if the config is not valid
// error will be returned instead.
func (npf *NopProcessorFactory) CreateTraceProcessor(
	_ *zap.Logger,
	nextConsumer consumer.TraceConsumerOld,
	_ configmodels.Processor,
) (component.TraceProcessorOld, error) {
	return &nopProcessor{nextTraceProcessor: nextConsumer}, nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
// If the processor type does not support metrics or if the config is not valid
// error will be returned instead.
func (npf *NopProcessorFactory) CreateMetricsProcessor(
	_ *zap.Logger,
	nextConsumer consumer.MetricsConsumerOld,
	_ configmodels.Processor,
) (component.MetricsProcessorOld, error) {
	return &nopProcessor{nextMetricsProcessor: nextConsumer}, nil
}
