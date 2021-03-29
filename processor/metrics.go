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
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/obsreport"
)

// Keys and stats for telemetry.
var (
	TagServiceNameKey, _   = tag.NewKey("service")
	TagProcessorNameKey, _ = tag.NewKey(obsreport.ProcessorKey)

	StatDroppedSpanCount = stats.Int64(
		"spans_dropped",
		"counts the number of spans dropped",
		stats.UnitDimensionless)

	StatTraceBatchesDroppedCount = stats.Int64(
		"trace_batches_dropped",
		"counts the number of trace batches dropped",
		stats.UnitDimensionless)
)

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
	droppedBatchesView := &view.View{
		Measure:     StatTraceBatchesDroppedCount,
		Description: "The number of span batches dropped.",
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
		droppedBatchesView,
		droppedSpansView,
	}

	return obsreport.ProcessorMetricViews("", legacyViews)
}
