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
	"bytes"
	"context"
	"errors"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	// TODO: once this repository has been transferred to the
	// official census-ecosystem location, update this import path.
	"github.com/orijtech/prometheus-go-metrics-exporter"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
)

var errBlankPrometheusAddress = errors.New("expecting a non-blank address to run the Prometheus metrics handler")

type prometheusExporter struct {
	name         string
	exporter     *prometheus.Exporter
	shutdownFunc func() error
}

func (pe *prometheusExporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pe *prometheusExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	ocmds := internaldata.MetricsToOC(md)
	for _, ocmd := range ocmds {
		merged := make(map[string]*metricspb.Metric)
		for _, metric := range ocmd.Metrics {
			merge(merged, metric)
		}
		for _, metric := range merged {
			_ = pe.exporter.ExportMetric(ctx, ocmd.Node, ocmd.Resource, metric)
		}
	}
	return nil
}

// The underlying exporter overwrites timeseries when there are conflicting metric signatures.
// Therefore, we need to merge timeseries that share a metric signature into a single metric before sending.
func merge(m map[string]*metricspb.Metric, metric *metricspb.Metric) {
	key := metricSignature(metric)
	current, ok := m[key]
	if !ok {
		m[key] = metric
		return
	}
	current.Timeseries = append(current.Timeseries, metric.Timeseries...)
}

// Unique identifier of a given promtheus metric
// Assumes label keys are always in the same order
func metricSignature(metric *metricspb.Metric) string {
	var buf bytes.Buffer
	buf.WriteString(metric.GetMetricDescriptor().GetName())
	labelKeys := metric.GetMetricDescriptor().GetLabelKeys()
	for _, labelKey := range labelKeys {
		buf.WriteString("-" + labelKey.Key)
	}
	return buf.String()
}

// Shutdown stops the exporter and is invoked during shutdown.
func (pe *prometheusExporter) Shutdown(context.Context) error {
	return pe.shutdownFunc()
}
