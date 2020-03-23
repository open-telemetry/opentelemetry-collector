// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreport

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

const (
	// Key used to identify processors in metrics and traces.
	ProcessorKey = "processor"

	// Key used to identify spans dropped by the Collector.
	DroppedSpansKey = "dropped_spans"

	// Key used to identify metric points dropped by the Collector.
	DroppedMetricPointsKey = "dropped_metric_points"
)

var (
	tagKeyProcessor, _ = tag.NewKey(ProcessorKey)

	processorPrefix = ProcessorKey + nameSep

	// Processor metrics. Any count of data items below is in the internal format
	// of the collector since processors only deal with internal format.
	mProcessorAcceptedSpans = stats.Int64(
		processorPrefix+AcceptedSpansKey,
		"Number of spans successfully pushed into the next component in the pipeline.",
		stats.UnitDimensionless)
	mProcessorRefusedSpans = stats.Int64(
		processorPrefix+RefusedSpansKey,
		"Number of spans that were rejected by the next component in the pipeline.",
		stats.UnitDimensionless)
	mProcessorDroppedSpans = stats.Int64(
		processorPrefix+DroppedSpansKey,
		"Number of spans that were dropped.",
		stats.UnitDimensionless)
	mProcessorAcceptedMetricPoints = stats.Int64(
		processorPrefix+AcceptedMetricPointsKey,
		"Number of metric points successfully pushed into the next component in the pipeline.",
		stats.UnitDimensionless)
	mProcessorRefusedMetricPoints = stats.Int64(
		processorPrefix+RefusedMetricPointsKey,
		"Number of metric points that were rejected by the next component in the pipeline.",
		stats.UnitDimensionless)
	mProcessorDroppedMetricPoints = stats.Int64(
		processorPrefix+DroppedMetricPointsKey,
		"Number of metric points that were dropped.",
		stats.UnitDimensionless)
)

// BuildProcessorCustomMetricName is used to be build a metric name following
// the standards used in the Collector. The configType should be the same
// value used to identify the type on the config.
func BuildProcessorCustomMetricName(configType, metric string) string {
	return buildComponentPrefix(processorPrefix, configType) + metric
}

// ProcessorMetricViews builds the metric views for custom metrics of processors.
func ProcessorMetricViews(configType string, legacyViews []*view.View) []*view.View {
	var allViews []*view.View
	if useLegacy {
		allViews = legacyViews
	}
	if useNew {
		for _, legacyView := range legacyViews {
			// Ignore any nil entry and views without measure or aggregation.
			// These can't be registered but some code registering legacy views may
			// ignore the errors.
			if legacyView == nil || legacyView.Measure == nil || legacyView.Aggregation == nil {
				continue
			}
			newView := *legacyView
			viewName := legacyView.Name
			if viewName == "" {
				viewName = legacyView.Measure.Name()
			}
			newView.Name = BuildProcessorCustomMetricName(configType, viewName)
			allViews = append(allViews, &newView)
		}
	}

	return allViews
}

// ProcessorContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ProcessorContext(
	ctx context.Context,
	processor string,
) context.Context {
	if useNew {
		ctx, _ = tag.New(
			ctx, tag.Upsert(tagKeyProcessor, processor, tag.WithTTL(tag.TTLNoPropagation)))
	}

	return ctx
}

// ProcessorTraceDataAccepted reports that the trace data was accepted.
func ProcessorTraceDataAccepted(
	processorCtx context.Context,
	td consumerdata.TraceData,
) {
	if useNew {
		stats.Record(
			processorCtx,
			mProcessorAcceptedSpans.M(int64(len(td.Spans))),
			mProcessorRefusedSpans.M(0),
			mProcessorDroppedSpans.M(0),
		)
	}
}

// ProcessorTraceDataRefused reports that the trace data was refused.
func ProcessorTraceDataRefused(
	processorCtx context.Context,
	td consumerdata.TraceData,
) {
	if useNew {
		stats.Record(
			processorCtx,
			mProcessorAcceptedSpans.M(0),
			mProcessorRefusedSpans.M(int64(len(td.Spans))),
			mProcessorDroppedSpans.M(0),
		)
	}
}

// ProcessorTraceDataDropped reports that the trace data was dropped.
func ProcessorTraceDataDropped(
	processorCtx context.Context,
	td consumerdata.TraceData,
) {
	if useNew {
		stats.Record(
			processorCtx,
			mProcessorAcceptedSpans.M(0),
			mProcessorRefusedSpans.M(0),
			mProcessorDroppedSpans.M(int64(len(td.Spans))),
		)
	}
}

// ProcessorMetricsDataAccepted reports that the metrics were accepted.
func ProcessorMetricsDataAccepted(
	processorCtx context.Context,
	md consumerdata.MetricsData,
) {
	if useNew {
		numPoints := calcNumPoints(md)
		stats.Record(
			processorCtx,
			mProcessorAcceptedMetricPoints.M(numPoints),
			mProcessorRefusedMetricPoints.M(0),
			mProcessorDroppedMetricPoints.M(0),
		)
	}
}

// ProcessorMetricsDataRefused reports that the metrics were refused.
func ProcessorMetricsDataRefused(
	processorCtx context.Context,
	md consumerdata.MetricsData,
) {
	if useNew {
		numPoints := calcNumPoints(md)
		stats.Record(
			processorCtx,
			mProcessorAcceptedMetricPoints.M(0),
			mProcessorRefusedMetricPoints.M(numPoints),
			mProcessorDroppedMetricPoints.M(0),
		)
	}
}

// ProcessorMetricsDataDropped reports that the metrics were dropped.
func ProcessorMetricsDataDropped(
	processorCtx context.Context,
	md consumerdata.MetricsData,
) {
	if useNew {
		numPoints := calcNumPoints(md)
		stats.Record(
			processorCtx,
			mProcessorAcceptedMetricPoints.M(0),
			mProcessorRefusedMetricPoints.M(0),
			mProcessorDroppedMetricPoints.M(numPoints),
		)
	}
}

func calcNumPoints(md consumerdata.MetricsData) int64 {
	numMetricPoints := 0
	for _, metric := range md.Metrics {
		for _, ts := range metric.GetTimeseries() {
			numMetricPoints += len(ts.GetPoints())
		}
	}
	return int64(numMetricPoints)
}
