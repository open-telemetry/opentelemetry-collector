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
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
)

// Deprecated: [v0.68.0] use processortest.NewNopCreateSettings.
func NewNopProcessorCreateSettings() processor.CreateSettings {
	return processor.CreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// Deprecated: [v0.68.0] use processortest.NewNopFactory.
func NewNopProcessorFactory() processor.Factory {
	return processor.NewFactory(
		"nop",
		func() component.Config { return &nopConfig{} },
		processor.WithTraces(createTracesProcessor, component.StabilityLevelStable),
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelStable),
		processor.WithLogs(createLogsProcessor, component.StabilityLevelStable),
	)
}

func createTracesProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Traces) (processor.Traces, error) {
	return nopProcessorInstance, nil
}

func createMetricsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return nopProcessorInstance, nil
}

func createLogsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Logs) (processor.Logs, error) {
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
