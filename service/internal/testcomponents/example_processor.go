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

func createTracesProcessor(_ context.Context, _ processor.CreateSettings, _ component.Config, nextConsumer consumer.Traces) (processor.Traces, error) {
	return &ExampleProcessor{Traces: nextConsumer}, nil
}

func createMetricsProcessor(_ context.Context, _ processor.CreateSettings, _ component.Config, nextConsumer consumer.Metrics) (processor.Metrics, error) {
	return &ExampleProcessor{Metrics: nextConsumer}, nil
}

func createLogsProcessor(_ context.Context, _ processor.CreateSettings, _ component.Config, nextConsumer consumer.Logs) (processor.Logs, error) {
	return &ExampleProcessor{Logs: nextConsumer}, nil
}

type ExampleProcessor struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
	Started bool
	Stopped bool
}

func (ep *ExampleProcessor) Start(_ context.Context, _ component.Host) error {
	ep.Started = true
	return nil
}

func (ep *ExampleProcessor) Shutdown(_ context.Context) error {
	ep.Stopped = true
	return nil
}

func (ep *ExampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
