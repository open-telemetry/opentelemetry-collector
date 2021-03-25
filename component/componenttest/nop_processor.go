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

package componenttest

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

// nopProcessorFactory is factory for nopProcessor.
type nopProcessorFactory struct{}

var nopProcessorFactoryInstance = &nopProcessorFactory{}

// NewNopProcessorFactory returns a component.ProcessorFactory that constructs nop exporters.
func NewNopProcessorFactory() component.ProcessorFactory {
	return nopProcessorFactoryInstance
}

// Type gets the type of the Processor config created by this factory.
func (f *nopProcessorFactory) Type() configmodels.Type {
	return "nop"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *nopProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &configmodels.ProcessorSettings{
		TypeVal: f.Type(),
	}
}

// CreateTracesProcessor implements component.ProcessorFactory interface.
func (f *nopProcessorFactory) CreateTracesProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ configmodels.Processor,
	_ consumer.Traces,
) (component.TracesProcessor, error) {
	return nopProcessorInstance, nil
}

// CreateMetricsProcessor implements component.ProcessorFactory interface.
func (f *nopProcessorFactory) CreateMetricsProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ configmodels.Processor,
	_ consumer.Metrics,
) (component.MetricsProcessor, error) {
	return nopProcessorInstance, nil
}

// CreateMetricsProcessor implements component.ProcessorFactory interface.
func (f *nopProcessorFactory) CreateLogsProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ configmodels.Processor,
	_ consumer.Logs,
) (component.LogsProcessor, error) {
	return nopProcessorInstance, nil
}

var nopProcessorInstance = &nopProcessor{
	Component: componenthelper.New(),
	Traces:    consumertest.NewTracesNop(),
	Metrics:   consumertest.NewMetricsNop(),
	Logs:      consumertest.NewLogsNop(),
}

// nopProcessor stores consumed traces and metrics for testing purposes.
type nopProcessor struct {
	component.Component
	consumer.Traces
	consumer.Metrics
	consumer.Logs
}

func (*nopProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}
