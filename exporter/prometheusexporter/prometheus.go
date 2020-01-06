// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusexporter

import (
	"context"
	"errors"

	// TODO: once this repository has been transferred to the
	// official census-ecosystem location, update this import path.
	"github.com/orijtech/prometheus-go-metrics-exporter"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"
)

var errBlankPrometheusAddress = errors.New("expecting a non-blank address to run the Prometheus metrics handler")

type prometheusExporter struct {
	name     string
	exporter *prometheus.Exporter
	shutdown exporterhelper.Shutdown
}

var _ consumer.MetricsConsumer = (*prometheusExporter)(nil)

func (pe *prometheusExporter) Start(host component.Host) error {
	return nil
}

func (pe *prometheusExporter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	for _, metric := range md.Metrics {
		_ = pe.exporter.ExportMetric(ctx, md.Node, md.Resource, metric)
	}
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (pe *prometheusExporter) Shutdown() error {
	return pe.shutdown()
}
