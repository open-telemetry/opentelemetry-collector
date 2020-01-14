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

// This file contains metrics to record dropped data via memory limiter,
// the package and its int wouldn't be necessary when proper dependencies are
// exposed via packages.

package memorylimiter

import (
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Keys and stats for telemetry.
var (
	TagExporterNameKey, _ = tag.NewKey("exporter")

	StatDroppedSpanCount = stats.Int64(
		"spans_dropped",
		"counts the number of spans dropped",
		stats.UnitDimensionless)

	StatDroppedMetricCount = stats.Int64(
		"metrics_dropped",
		"counts the number of metrics dropped",
		stats.UnitDimensionless)
)

var initOnce sync.Once

func initMetrics() {
	initOnce.Do(func() {
		tagKeys := []tag.Key{
			TagExporterNameKey,
		}
		droppedSpanBatchesView := &view.View{
			Name:        "batches_dropped",
			Measure:     StatDroppedSpanCount,
			Description: "The number of span batches dropped.",
			TagKeys:     tagKeys,
			Aggregation: view.Count(),
		}
		droppedSpansView := &view.View{
			Name:        StatDroppedSpanCount.Name(),
			Measure:     StatDroppedSpanCount,
			Description: "The number of spans dropped.",
			TagKeys:     tagKeys,
			Aggregation: view.Sum(),
		}

		droppedMetricBatchesView := &view.View{
			Name:        "batches_dropped",
			Measure:     StatDroppedMetricCount,
			Description: "The number of metric batches dropped.",
			TagKeys:     tagKeys,
			Aggregation: view.Count(),
		}
		droppedMetricsView := &view.View{
			Name:        StatDroppedMetricCount.Name(),
			Measure:     StatDroppedMetricCount,
			Description: "The number of metrics dropped.",
			TagKeys:     tagKeys,
			Aggregation: view.Sum(),
		}

		view.Register(droppedSpanBatchesView, droppedSpansView, droppedMetricBatchesView, droppedMetricsView)
	})
}

// statsTagsForBatch creates a tag.Mutator that can be used to add a metric
// label with the processorName to context.Context via stats.RecordWithTags
// function. This ensures uniformity of labels for the metrics.
func statsTagsForBatch(processorName string) []tag.Mutator {
	statsTags := []tag.Mutator{
		tag.Upsert(TagExporterNameKey, processorName),
	}

	return statsTags
}
