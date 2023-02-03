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

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const procType = "exampleprocessor"

// ExampleProcessorFactory is factory for ExampleProcessor.
var ExampleProcessorFactory = processor.NewFactory(
	procType,
	createDefaultConfig,
	processor.WithTraces(createTracesProcessor, component.StabilityLevelDevelopment),
	processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	processor.WithLogs(createLogsProcessor, component.StabilityLevelDevelopment))

// CreateDefaultConfig creates the default configuration for the Processor.
func createDefaultConfig() component.Config {
	return &struct{}{}
}

func createTracesProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumer.Traces) (processor.Traces, error) {
	return &ExampleProcessor{
		ConsumeTracesFunc: nextConsumer.ConsumeTraces,
		mutatesData:       set.ID.Name() == "mutate",
	}, nil
}

func createMetricsProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumer.Metrics) (processor.Metrics, error) {
	return &ExampleProcessor{
		ConsumeMetricsFunc: nextConsumer.ConsumeMetrics,
		mutatesData:        set.ID.Name() == "mutate",
	}, nil
}

func createLogsProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumer.Logs) (processor.Logs, error) {
	return &ExampleProcessor{
		ConsumeLogsFunc: nextConsumer.ConsumeLogs,
		mutatesData:     set.ID.Name() == "mutate",
	}, nil
}

type ExampleProcessor struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	mutatesData bool
}

func (ep *ExampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: ep.mutatesData}
}
