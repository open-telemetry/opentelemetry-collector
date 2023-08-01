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

package processortest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
)

// NewUnhealthyProcessorCreateSettings returns a new nop settings for Create*Processor functions.
func NewUnhealthyProcessorCreateSettings() processor.CreateSettings {
	return processor.CreateSettings{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewUnhealthyProcessorFactory returns a component.ProcessorFactory that constructs nop processors.
func NewUnhealthyProcessorFactory() processor.Factory {
	return processor.NewFactory(
		"unhealthy",
		func() component.Config {
			return &struct{}{}
		},
		processor.WithTraces(createUnhealthyTracesProcessor, component.StabilityLevelStable),
		processor.WithMetrics(createUnhealthyMetricsProcessor, component.StabilityLevelStable),
		processor.WithLogs(createUnhealthyLogsProcessor, component.StabilityLevelStable),
	)
}

func createUnhealthyTracesProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Traces) (processor.Traces, error) {
	return unhealthyProcessorInstance, nil
}

func createUnhealthyMetricsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return unhealthyProcessorInstance, nil
}

func createUnhealthyLogsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Logs) (processor.Logs, error) {
	return unhealthyProcessorInstance, nil
}

var unhealthyProcessorInstance = &unhealthyProcessor{
	Consumer: consumertest.NewNop(),
}

// unhealthyProcessor stores consumed traces and metrics for testing purposes.
type unhealthyProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func (unhealthyProcessor) Start(ctx context.Context, host component.Host) error {
	go func() {
		evt, _ := component.NewStatusEvent(component.StatusError)
		host.ReportComponentStatus(evt)
	}()
	return nil
}
