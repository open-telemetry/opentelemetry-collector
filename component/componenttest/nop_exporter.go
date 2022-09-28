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
	"go.opentelemetry.io/collector/consumer/consumertest"
)

// NewNopExporterCreateSettings returns a new nop settings for Create*Exporter functions.
func NewNopExporterCreateSettings() component.ExporterCreateSettings {
	return component.ExporterCreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopExporterConfig struct {
	config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// NewNopExporterFactory returns a component.ExporterFactory that constructs nop exporters.
func NewNopExporterFactory() component.ExporterFactory {
	return component.NewExporterFactory(
		"nop",
		func() config.Exporter {
			return &nopExporterConfig{
				ExporterSettings: config.NewExporterSettings(config.NewComponentID("nop")),
			}
		},
		component.WithTracesExporter(createTracesExporter, component.StabilityLevelStable),
		component.WithMetricsExporter(createMetricsExporter, component.StabilityLevelStable),
		component.WithLogsExporter(createLogsExporter, component.StabilityLevelStable),
	)
}

func createTracesExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.TracesExporter, error) {
	return nopExporterInstance, nil
}

func createMetricsExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.MetricsExporter, error) {
	return nopExporterInstance, nil
}

func createLogsExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.LogsExporter, error) {
	return nopExporterInstance, nil
}

var nopExporterInstance = &nopExporter{
	Consumer: consumertest.NewNop(),
}

// nopExporter stores consumed traces and metrics for testing purposes.
type nopExporter struct {
	nopComponent
	consumertest.Consumer
}
