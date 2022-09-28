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

// NewNopProcessorCreateSettings returns a new nop settings for Create*Processor functions.
func NewNopProcessorCreateSettings() component.ProcessorCreateSettings {
	return component.ProcessorCreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopProcessorConfig struct {
	config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// NewNopProcessorFactory returns a component.ProcessorFactory that constructs nop processors.
func NewNopProcessorFactory() component.ProcessorFactory {
	return component.NewProcessorFactory(
		"nop",
		func() config.Processor {
			return &nopProcessorConfig{
				ProcessorSettings: config.NewProcessorSettings(config.NewComponentID("nop")),
			}
		},
		component.WithTracesProcessor(createTracesProcessor, component.StabilityLevelStable),
		component.WithMetricsProcessor(createMetricsProcessor, component.StabilityLevelStable),
		component.WithLogsProcessor(createLogsProcessor, component.StabilityLevelStable),
	)
}

func createTracesProcessor(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Traces) (component.TracesProcessor, error) {
	return nopProcessorInstance, nil
}

func createMetricsProcessor(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Metrics) (component.MetricsProcessor, error) {
	return nopProcessorInstance, nil
}

func createLogsProcessor(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Logs) (component.LogsProcessor, error) {
	return nopProcessorInstance, nil
}

var nopProcessorInstance = &nopProcessor{
	Consumer: consumertest.NewNop(),
}

// nopProcessor stores consumed traces and metrics for testing purposes.
type nopProcessor struct {
	nopComponent
	consumertest.Consumer
}
