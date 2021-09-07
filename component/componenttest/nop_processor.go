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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

// NewNopProcessorCreateSettings returns a new nop settings for Create*Processor functions.
func NewNopProcessorCreateSettings() component.ProcessorCreateSettings {
	return component.ProcessorCreateSettings{
		TelemetryCreateSettings: NewNopTelemetryCreateSettings(),
		BuildInfo:               component.DefaultBuildInfo(),
	}
}

type nopProcessorConfig struct {
	config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// nopProcessorFactory is factory for nopProcessor.
type nopProcessorFactory struct {
	component.BaseProcessorFactory
}

var nopProcessorFactoryInstance = &nopProcessorFactory{}

// NewNopProcessorFactory returns a component.ProcessorFactory that constructs nop processors.
func NewNopProcessorFactory() component.ProcessorFactory {
	return nopProcessorFactoryInstance
}

// Type gets the type of the Processor config created by this factory.
func (f *nopProcessorFactory) Type() config.Type {
	return "nop"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *nopProcessorFactory) CreateDefaultConfig() config.Processor {
	return &nopProcessorConfig{
		ProcessorSettings: config.NewProcessorSettings(config.NewID("nop")),
	}
}

// CreateTracesProcessor implements component.ProcessorFactory interface.
func (f *nopProcessorFactory) CreateTracesProcessor(
	_ context.Context,
	_ component.ProcessorCreateSettings,
	_ config.Processor,
	_ consumer.Traces,
) (component.TracesProcessor, error) {
	return nopProcessorInstance, nil
}

// CreateMetricsProcessor implements component.ProcessorFactory interface.
func (f *nopProcessorFactory) CreateMetricsProcessor(
	_ context.Context,
	_ component.ProcessorCreateSettings,
	_ config.Processor,
	_ consumer.Metrics,
) (component.MetricsProcessor, error) {
	return nopProcessorInstance, nil
}

// CreateLogsProcessor implements component.ProcessorFactory interface.
func (f *nopProcessorFactory) CreateLogsProcessor(
	_ context.Context,
	_ component.ProcessorCreateSettings,
	_ config.Processor,
	_ consumer.Logs,
) (component.LogsProcessor, error) {
	return nopProcessorInstance, nil
}

var nopProcessorInstance = &nopProcessor{
	Component: componenthelper.New(),
	Consumer:  consumertest.NewNop(),
}

// nopProcessor stores consumed traces and metrics for testing purposes.
type nopProcessor struct {
	component.Component
	consumertest.Consumer
}
