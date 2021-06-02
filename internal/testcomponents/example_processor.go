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

package testcomponents

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// ExampleProcessorCfg is for testing purposes. We are defining an example config and factory
// for "exampleprocessor" processor type.
type ExampleProcessorCfg struct {
	config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting             string                   `mapstructure:"extra"`
	ExtraMapSetting          map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting         []string                 `mapstructure:"extra_list"`
}

const procType = "exampleprocessor"

// ExampleProcessorFactory is factory for exampleProcessor.
var ExampleProcessorFactory = processorhelper.NewFactory(
	procType,
	createDefaultConfig,
	processorhelper.WithTraces(createTracesProcessor),
	processorhelper.WithMetrics(createMetricsProcessor),
	processorhelper.WithLogs(createLogsProcessor))

// CreateDefaultConfig creates the default configuration for the Processor.
func createDefaultConfig() config.Processor {
	return &ExampleProcessorCfg{
		ProcessorSettings: config.NewProcessorSettings(config.NewID(procType)),
		ExtraSetting:      "some export string",
		ExtraMapSetting:   nil,
		ExtraListSetting:  nil,
	}
}

func createTracesProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Traces) (component.TracesProcessor, error) {
	return &exampleProcessor{Traces: nextConsumer}, nil
}

func createMetricsProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Metrics) (component.MetricsProcessor, error) {
	return &exampleProcessor{Metrics: nextConsumer}, nil
}

func createLogsProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Logs) (component.LogsProcessor, error) {
	return &exampleProcessor{Logs: nextConsumer}, nil
}

type exampleProcessor struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
}

func (ep *exampleProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (ep *exampleProcessor) Shutdown(_ context.Context) error {
	return nil
}

func (ep *exampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
