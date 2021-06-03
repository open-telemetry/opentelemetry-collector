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

package prometheusexporter

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "prometheus"
)

// NewFactory creates a new Prometheus exporter factory.
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithMetrics(createMetricsExporter))
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		ConstLabels:      map[string]string{},
		SendTimestamps:   false,
		MetricExpiration: time.Minute * 5,
	}
}

func createMetricsExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.MetricsExporter, error) {
	pcfg := cfg.(*Config)

	prometheus, err := newPrometheusExporter(pcfg, set.Logger)
	if err != nil {
		return nil, err
	}

	exporter, err := exporterhelper.NewMetricsExporter(
		cfg,
		set.Logger,
		prometheus.ConsumeMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithStart(prometheus.Start),
		exporterhelper.WithShutdown(prometheus.Shutdown),
		exporterhelper.WithResourceToTelemetryConversion(pcfg.ResourceToTelemetrySettings),
	)
	if err != nil {
		return nil, err
	}

	return &wrapMetricsExpoter{
		MetricsExporter: exporter,
		exporter:        prometheus,
	}, nil
}

type wrapMetricsExpoter struct {
	component.MetricsExporter
	exporter *prometheusExporter
}
