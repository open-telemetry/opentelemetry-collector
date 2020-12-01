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

package processor

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/translator/conventions"
)

// Keys and stats for telemetry.
var (
	TagServiceNameKey, _   = tag.NewKey("service")
	TagProcessorNameKey, _ = tag.NewKey(obsreport.ProcessorKey)

	StatReceivedSpanCount = stats.Int64(
		"spans_received",
		"counts the number of spans received",
		stats.UnitDimensionless)
	StatDroppedSpanCount = stats.Int64(
		"spans_dropped",
		"counts the number of spans dropped",
		stats.UnitDimensionless)

	StatTraceBatchesDroppedCount = stats.Int64(
		"trace_batches_dropped",
		"counts the number of trace batches dropped",
		stats.UnitDimensionless)
)

// SpanCountStats represents span count stats grouped by service if DETAILED telemetry level is set,
// otherwise only overall span count is stored in serviceSpansCounts.
type SpanCountStats struct {
	serviceSpansCounts map[string]int
	allSpansCount      int
	isDetailed         bool
}

func NewSpanCountStats(td pdata.Traces) *SpanCountStats {
	scm := &SpanCountStats{
		allSpansCount: td.SpanCount(),
	}
	if serviceTagsEnabled() {
		scm.serviceSpansCounts = spanCountByResourceStringAttribute(td, conventions.AttributeServiceName)
		scm.isDetailed = true
	}
	return scm
}

func (scm *SpanCountStats) GetAllSpansCount() int {
	return scm.allSpansCount
}

// MetricTagKeys returns the metric tag keys according to the given telemetry level.
func MetricTagKeys() []tag.Key {
	return []tag.Key{
		TagProcessorNameKey,
		TagServiceNameKey,
	}
}

// MetricViews return the metrics views according to given telemetry level.
func MetricViews() []*view.View {
	tagKeys := MetricTagKeys()
	// There are some metrics enabled, return the views.
	receivedBatchesView := &view.View{
		Name:        "batches_received",
		Measure:     StatReceivedSpanCount,
		Description: "The number of span batches received.",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
	}
	droppedBatchesView := &view.View{
		Measure:     StatTraceBatchesDroppedCount,
		Description: "The number of span batches dropped.",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}
	receivedSpansView := &view.View{
		Name:        StatReceivedSpanCount.Name(),
		Measure:     StatReceivedSpanCount,
		Description: "The number of spans received.",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}
	droppedSpansView := &view.View{
		Name:        StatDroppedSpanCount.Name(),
		Measure:     StatDroppedSpanCount,
		Description: "The number of spans dropped.",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	legacyViews := []*view.View{
		receivedBatchesView,
		droppedBatchesView,
		receivedSpansView,
		droppedSpansView,
	}

	return obsreport.ProcessorMetricViews("", legacyViews)
}

// RecordsSpanCountMetrics reports span count metrics for specified measure.
func RecordsSpanCountMetrics(ctx context.Context, scm *SpanCountStats, measure *stats.Int64Measure) {
	if scm.isDetailed {
		for serviceName, spanCount := range scm.serviceSpansCounts {
			statsTags := []tag.Mutator{tag.Insert(TagServiceNameKey, serviceName)}
			_ = stats.RecordWithTags(ctx, statsTags, measure.M(int64(spanCount)))
		}
		return
	}

	stats.Record(ctx, measure.M(int64(scm.allSpansCount)))
}

func serviceTagsEnabled() bool {
	level := configtelemetry.GetMetricsLevelFlagValue()
	return level == configtelemetry.LevelDetailed
}

// spanCountByResourceStringAttribute calculates the number of spans by resource specified by
// provided string attribute attrKey.
func spanCountByResourceStringAttribute(td pdata.Traces, attrKey string) map[string]int {
	spanCounts := make(map[string]int)

	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		var attrStringVal string
		if attrVal, ok := rs.Resource().Attributes().Get(attrKey); ok {
			attrStringVal = attrVal.StringVal()
		}
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			spanCounts[attrStringVal] += ilss.At(j).Spans().Len()
		}
	}
	return spanCounts
}
