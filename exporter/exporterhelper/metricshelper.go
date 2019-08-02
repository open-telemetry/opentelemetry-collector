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

	"github.com/open-telemetry/opentelemetry-service/observability"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter"
)

// PushMetricsData is a helper function that is similar to ConsumeMetricsData but also returns
// the number of dropped metrics.
type PushMetricsData func(ctx context.Context, td consumerdata.MetricsData) (droppedMetrics int, err error)

type metricsExporter struct {
	exporterName    string
	pushMetricsData PushMetricsData
}

var _ (exporter.MetricsExporter) = (*metricsExporter)(nil)

func (me *metricsExporter) MetricsExportFormat() string {
	return me.exporterName
}

func (me *metricsExporter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	exporterCtx := observability.ContextWithExporterName(ctx, me.exporterName)
	_, err := me.pushMetricsData(exporterCtx, md)
	return err
}

// NewMetricsExporter creates an MetricsExporter that can record metrics and can wrap every request with a Span.
// If no options are passed it just adds the exporter format as a tag in the Context.
// TODO: Add support for recordMetrics.
// TODO: Add support for retries.
func NewMetricsExporter(exporterName string, pushMetricsData PushMetricsData, options ...ExporterOption) (exporter.MetricsExporter, error) {
	if exporterName == "" {
		return nil, errEmptyExporterName
	}

	if pushMetricsData == nil {
		return nil, errNilPushMetricsData
	}

	opts := newExporterOptions(options...)

	if opts.spanName != "" {
		pushMetricsData = pushMetricsDataWithSpan(pushMetricsData, opts.spanName)
	}

	return &metricsExporter{
		exporterName:    exporterName,
		pushMetricsData: pushMetricsData,
	}, nil
}

func pushMetricsDataWithSpan(next PushMetricsData, spanName string) PushMetricsData {
	return func(ctx context.Context, md consumerdata.MetricsData) (int, error) {
		ctx, span := trace.StartSpan(ctx, spanName)
		defer span.End()
		// Call next stage.
		droppedMetrics, err := next(ctx, md)
		if span.IsRecordingEvents() {
			span.AddAttributes(
				trace.Int64Attribute(numReceivedMetricsAttribute, int64(len(md.Metrics))),
				trace.Int64Attribute(numDroppedMetricsAttribute, int64(droppedMetrics)),
			)
			if err != nil {
				span.SetStatus(errToStatus(err))
			}
		}
		return droppedMetrics, err
	}
}
