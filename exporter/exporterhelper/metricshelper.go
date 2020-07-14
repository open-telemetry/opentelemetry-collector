// Copyright The OpenTelemetry Authors
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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/obsreport"
)

// PushMetricsDataOld is a helper function that is similar to ConsumeMetricsData but also returns
// the number of dropped metrics.
type PushMetricsDataOld func(ctx context.Context, td consumerdata.MetricsData) (droppedTimeSeries int, err error)

type metricsExporterOld struct {
	baseExporter
	pushMetricsData PushMetricsDataOld
}

func (me *metricsExporterOld) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	exporterCtx := obsreport.ExporterContext(ctx, me.exporterFullName)
	_, err := me.pushMetricsData(exporterCtx, md)
	return err
}

// NewMetricsExporterOld creates an MetricsExporter that can record metrics and can wrap every request with a Span.
// If no internalOptions are passed it just adds the exporter format as a tag in the Context.
// TODO: Add support for retries.
func NewMetricsExporterOld(config configmodels.Exporter, pushMetricsData PushMetricsDataOld, options ...ExporterOption) (component.MetricsExporterOld, error) {
	if config == nil {
		return nil, errNilConfig
	}

	if pushMetricsData == nil {
		return nil, errNilPushMetricsData
	}

	pushMetricsData = pushMetricsWithObservabilityOld(pushMetricsData, config.Name())

	return &metricsExporterOld{
		baseExporter:    newBaseExporter(config.Name(), options...),
		pushMetricsData: pushMetricsData,
	}, nil
}

func pushMetricsWithObservabilityOld(next PushMetricsDataOld, exporterName string) PushMetricsDataOld {
	return func(ctx context.Context, md consumerdata.MetricsData) (int, error) {
		ctx = obsreport.StartMetricsExportOp(ctx, exporterName)
		numDroppedTimeSeries, err := next(ctx, md)

		// TODO: this is not ideal: it should come from the next function itself.
		// 	temporarily loading it from internal format. Once full switch is done
		// 	to new metrics will remove this.
		numReceivedTimeSeries, numPoints := pdatautil.TimeseriesAndPointCount(md)

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

// PushMetricsData is a helper function that is similar to ConsumeMetricsData but also returns
// the number of dropped metrics.
type PushMetricsData func(ctx context.Context, md pdata.Metrics) (droppedTimeSeries int, err error)

type metricsExporter struct {
	baseExporter
	pushMetricsData PushMetricsData
}

func (me *metricsExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	exporterCtx := obsreport.ExporterContext(ctx, me.exporterFullName)
	_, err := me.pushMetricsData(exporterCtx, md)
	return err
}

// NewMetricsExporter creates an MetricsExporter that can record metrics and can wrap every request with a Span.
// If no internalOptions are passed it just adds the exporter format as a tag in the Context.
// TODO: Add support for retries.
func NewMetricsExporter(config configmodels.Exporter, pushMetricsData PushMetricsData, options ...ExporterOption) (component.MetricsExporter, error) {
	if config == nil {
		return nil, errNilConfig
	}

	if pushMetricsData == nil {
		return nil, errNilPushMetricsData
	}

	pushMetricsData = pushMetricsWithObservability(pushMetricsData, config.Name())

	return &metricsExporter{
		baseExporter:    newBaseExporter(config.Name(), options...),
		pushMetricsData: pushMetricsData,
	}, nil
}

func pushMetricsWithObservability(next PushMetricsData, exporterName string) PushMetricsData {
	return func(ctx context.Context, md pdata.Metrics) (int, error) {
		ctx = obsreport.StartMetricsExportOp(ctx, exporterName)
		numDroppedMetrics, err := next(ctx, md)

		// TODO: this is not ideal: it should come from the next function itself.
		// 	temporarily loading it from internal format. Once full switch is done
		// 	to new metrics will remove this.
		numReceivedMetrics, numPoints := pdatautil.MetricAndDataPointCount(md)

		obsreport.EndMetricsExportOp(ctx, numPoints, numReceivedMetrics, numDroppedMetrics, err)
		return numReceivedMetrics, err
	}
}
