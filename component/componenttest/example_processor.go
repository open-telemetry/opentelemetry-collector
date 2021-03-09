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
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// ExampleProcessorCfg is for testing purposes. We are defining an example config and factory
// for "exampleprocessor" processor type.
type ExampleProcessorCfg struct {
	configmodels.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
	ExtraMapSetting                map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting               []string                 `mapstructure:"extra_list"`
}

// ExampleProcessorFactory is factory for ExampleProcessor.
type ExampleProcessorFactory struct {
}

// Type gets the type of the Processor config created by this factory.
func (f *ExampleProcessorFactory) Type() configmodels.Type {
	return "exampleprocessor"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *ExampleProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &ExampleProcessorCfg{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: f.Type(),
			NameVal: string(f.Type()),
		},
		ExtraSetting:     "some export string",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *ExampleProcessorFactory) CreateTracesProcessor(ctx context.Context, params component.ProcessorCreateParams, cfg configmodels.Processor, nextConsumer consumer.TracesConsumer) (component.TracesProcessor, error) {
	return &ExampleProcessor{nextTraces: nextConsumer}, nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *ExampleProcessorFactory) CreateMetricsProcessor(ctx context.Context, params component.ProcessorCreateParams, cfg configmodels.Processor, nextConsumer consumer.MetricsConsumer) (component.MetricsProcessor, error) {
	return &ExampleProcessor{nextMetrics: nextConsumer}, nil
}

func (f *ExampleProcessorFactory) CreateLogsProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ configmodels.Processor,
	nextConsumer consumer.LogsConsumer,
) (component.LogsProcessor, error) {
	return &ExampleProcessor{nextLogs: nextConsumer}, nil
}

type ExampleProcessor struct {
	nextTraces  consumer.TracesConsumer
	nextMetrics consumer.MetricsConsumer
	nextLogs    consumer.LogsConsumer
}

func (ep *ExampleProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (ep *ExampleProcessor) Shutdown(_ context.Context) error {
	return nil
}

func (ep *ExampleProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

func (ep *ExampleProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return ep.nextTraces.ConsumeTraces(ctx, td)
}

func (ep *ExampleProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return ep.nextMetrics.ConsumeMetrics(ctx, md)
}

func (ep *ExampleProcessor) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	return ep.nextLogs.ConsumeLogs(ctx, ld)
}
