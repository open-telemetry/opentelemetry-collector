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

package fileexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
)

const (
	// The value of "type" key in configuration.
	typeStr = "file"
)

// NewFactory creates a factory for OTLP exporter.
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithTraces(createTracesExporter),
		exporterhelper.WithMetrics(createMetricsExporter),
		exporterhelper.WithLogs(createLogsExporter))
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
	}
}

func createTracesExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.TracesExporter, error) {
	fe := exporters.GetOrAdd(cfg, func() component.Component {
		return &fileExporter{path: cfg.(*Config).Path}
	})
	return exporterhelper.NewTracesExporter(
		cfg,
		set.Logger,
		fe.Unwrap().(*fileExporter).ConsumeTraces,
		exporterhelper.WithStart(fe.Start),
		exporterhelper.WithShutdown(fe.Shutdown),
	)
}

func createMetricsExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.MetricsExporter, error) {
	fe := exporters.GetOrAdd(cfg, func() component.Component {
		return &fileExporter{path: cfg.(*Config).Path}
	})
	return exporterhelper.NewMetricsExporter(
		cfg,
		set.Logger,
		fe.Unwrap().(*fileExporter).ConsumeMetrics,
		exporterhelper.WithStart(fe.Start),
		exporterhelper.WithShutdown(fe.Shutdown),
	)
}

func createLogsExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.LogsExporter, error) {
	fe := exporters.GetOrAdd(cfg, func() component.Component {
		return &fileExporter{path: cfg.(*Config).Path}
	})
	return exporterhelper.NewLogsExporter(
		cfg,
		set.Logger,
		fe.Unwrap().(*fileExporter).ConsumeLogs,
		exporterhelper.WithStart(fe.Start),
		exporterhelper.WithShutdown(fe.Shutdown),
	)
}

// This is the map of already created File exporters for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exporters = sharedcomponent.NewSharedComponents()
