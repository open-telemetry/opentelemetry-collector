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

package processortest // import "go.opentelemetry.io/collector/processor/processortest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "nop"

// NewNopCreateSettings returns a new nop settings for Create* functions.
func NewNopCreateSettings() processor.CreateSettings {
	return processor.CreateSettings{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a component.ProcessorFactory that constructs nop processors.
func NewNopFactory() processor.Factory {
	return processor.NewFactory(
		"nop",
		func() component.Config { return &nopConfig{} },
		processor.WithTraces(createTracesProcessor, component.StabilityLevelStable),
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelStable),
		processor.WithLogs(createLogsProcessor, component.StabilityLevelStable),
	)
}

func createTracesProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Traces) (processor.Traces, error) {
	return nopInstance, nil
}

func createMetricsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return nopInstance, nil
}

func createLogsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Logs) (processor.Logs, error) {
	return nopInstance, nil
}

type nopConfig struct{}

var nopInstance = &nopProcessor{
	Consumer: consumertest.NewNop(),
}

// nopProcessor stores consumed traces and metrics for testing purposes.
type nopProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

// NewNopBuilder returns a processor.Builder that constructs nop receivers.
func NewNopBuilder() *processor.Builder {
	nopFactory := NewNopFactory()
	return processor.NewBuilder(
		map[component.ID]component.Config{component.NewID(typeStr): nopFactory.CreateDefaultConfig()},
		map[component.Type]processor.Factory{typeStr: nopFactory})
}
