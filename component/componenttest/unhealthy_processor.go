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

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

// NewUnhealthyProcessorCreateSettings returns a new nop settings for Create*Processor functions.
func NewUnhealthyProcessorCreateSettings() component.ProcessorCreateSettings {
	return component.ProcessorCreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type unhealthyProcessorConfig struct {
	config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// NewUnhealthyProcessorFactory returns a component.ProcessorFactory that constructs nop processors.
func NewUnhealthyProcessorFactory() component.ProcessorFactory {
	return component.NewProcessorFactory(
		"unhealthy",
		func() component.ProcessorConfig {
			return &unhealthyProcessorConfig{
				ProcessorSettings: config.NewProcessorSettings(component.NewID("nop")),
			}
		},
		component.WithTracesProcessor(createUnhealthyTracesProcessor, component.StabilityLevelStable),
		component.WithMetricsProcessor(createUnhealthyMetricsProcessor, component.StabilityLevelStable),
		component.WithLogsProcessor(createUnhealthyLogsProcessor, component.StabilityLevelStable),
	)
}

func createUnhealthyTracesProcessor(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Traces) (component.TracesProcessor, error) {
	return unhealthyProcessorInstance, nil
}

func createUnhealthyMetricsProcessor(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Metrics) (component.MetricsProcessor, error) {
	return unhealthyProcessorInstance, nil
}

func createUnhealthyLogsProcessor(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Logs) (component.LogsProcessor, error) {
	return unhealthyProcessorInstance, nil
}

var unhealthyProcessorInstance = &unhealthyProcessor{
	Consumer: consumertest.NewNop(),
}

// unhealthyProcessor stores consumed traces and metrics for testing purposes.
type unhealthyProcessor struct {
	nopComponent
	consumertest.Consumer
}

func (unhealthyProcessor) Start(ctx context.Context, host component.Host) error {
	go func() {
		evt, _ := component.NewStatusEvent(component.StatusError)
		host.ReportComponentStatus(evt)
	}()
	return nil
}
