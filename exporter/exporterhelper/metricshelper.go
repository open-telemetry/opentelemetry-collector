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

package exporterhelper

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
)

// PushMetricsData is a helper function that is similar to ConsumeMetricsData but also returns
// the number of dropped metrics.
type PushMetricsData func(ctx context.Context, td consumerdata.MetricsData) (droppedTimeSeries int, err error)

type metricsExporter struct {
	exporterFullName string
	pushMetricsData  PushMetricsData
	shutdown         Shutdown
}

var _ (exporter.MetricsExporter) = (*metricsExporter)(nil)

func (me *metricsExporter) Start(host component.Host) error {
	return nil
}

func (me *metricsExporter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	exporterCtx := obsreport.ExporterContext(ctx, me.exporterFullName)
	_, err := me.pushMetricsData(exporterCtx, md)
	return err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (me *metricsExporter) Shutdown() error {
	return me.shutdown()
}

// NewMetricsExporter creates an MetricsExporter that can record metrics and can wrap every request with a Span.
// If no options are passed it just adds the exporter format as a tag in the Context.
// TODO: Add support for retries.
func NewMetricsExporter(config configmodels.Exporter, pushMetricsData PushMetricsData, options ...ExporterOption) (exporter.MetricsExporter, error) {
	if config == nil {
		return nil, errNilConfig
	}

	if pushMetricsData == nil {
		return nil, errNilPushMetricsData
	}

	opts := newExporterOptions(options...)

	pushMetricsData = pushMetricsWithObservability(pushMetricsData, config.Name())

	// The default shutdown method always returns nil.
	if opts.shutdown == nil {
		opts.shutdown = func() error { return nil }
	}

	return &metricsExporter{
		exporterFullName: config.Name(),
		pushMetricsData:  pushMetricsData,
		shutdown:         opts.shutdown,
	}, nil
}

func pushMetricsWithObservability(next PushMetricsData, exporterName string) PushMetricsData {
	return func(ctx context.Context, md consumerdata.MetricsData) (int, error) {
		ctx = obsreport.StartMetricsExportOp(ctx, exporterName)
		numDroppedTimeSeries, err := next(ctx, md)

		// TODO: this is not ideal: it should come from the next function itself.
		// 	temporarily loading it from internal format. Once full switch is done
		// 	to new metrics will remove this.
		numReceivedTimeSeries, numPoints := measureMetricsExport(md)

		obsreport.EndMetricsExportOp(ctx, numPoints, numReceivedTimeSeries, numDroppedTimeSeries, err)
		return numDroppedTimeSeries, err
	}
}

// NumTimeSeries returns the number of timeseries in a MetricsData.
func NumTimeSeries(md consumerdata.MetricsData) int {
	receivedTimeSeries := 0
	for _, metric := range md.Metrics {
		receivedTimeSeries += len(metric.GetTimeseries())
	}
	return receivedTimeSeries
}

func measureMetricsExport(md consumerdata.MetricsData) (int, int) {
	numTimeSeries := 0
	numPoints := 0
	for _, metric := range md.Metrics {
		tss := metric.GetTimeseries()
		numTimeSeries += len(metric.GetTimeseries())
		for _, ts := range tss {
			numPoints += len(ts.GetPoints())
		}
	}
	return numTimeSeries, numPoints
}
