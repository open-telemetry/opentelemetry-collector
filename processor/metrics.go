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

package processor

import (
	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/open-telemetry/opentelemetry-collector/internal/collector/telemetry"
)

// Keys and stats for telemetry.
var (
	TagSourceFormatKey, _  = tag.NewKey("source_format")
	TagServiceNameKey, _   = tag.NewKey("service")
	TagProcessorNameKey, _ = tag.NewKey("processor")

	StatReceivedSpanCount = stats.Int64(
		"spans_received",
		"counts the number of spans received",
		stats.UnitDimensionless)
	StatDroppedSpanCount = stats.Int64(
		"spans_dropped",
		"counts the number of spans dropped",
		stats.UnitDimensionless)
	StatBadBatchDroppedSpanCount = stats.Int64(
		"bad_batch_spans_dropped",
		"counts the number of spans dropped due to being in bad batches",
		stats.UnitDimensionless)
)

// MetricTagKeys returns the metric tag keys according to the given telemetry level.
func MetricTagKeys(level telemetry.Level) []tag.Key {
	var tagKeys []tag.Key
	switch level {
	case telemetry.Detailed:
		tagKeys = append(tagKeys, TagServiceNameKey)
		fallthrough
	case telemetry.Normal:
		tagKeys = append(tagKeys, TagSourceFormatKey)
		fallthrough
	case telemetry.Basic:
		tagKeys = append(tagKeys, TagProcessorNameKey)
	default:
		return nil
	}

	return tagKeys
}

// MetricViews return the metrics views according to given telemetry level.
func MetricViews(level telemetry.Level) []*view.View {
	tagKeys := MetricTagKeys(level)
	if tagKeys == nil {
		return nil
	}

	// There are some metrics enabled, return the views.
	receivedBatchesView := &view.View{
		Name:        "batches_received",
		Measure:     StatReceivedSpanCount,
		Description: "The number of span batches received.",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
	}
	droppedBatchesView := &view.View{
		Name:        "batches_dropped",
		Measure:     StatDroppedSpanCount,
		Description: "The number of span batches dropped.",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
	}
	droppedBadBatchesView := &view.View{
		Name:        "bad_batches_dropped",
		Measure:     StatBadBatchDroppedSpanCount,
		Description: "The number of span batches with bad data that were dropped.",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
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
	droppedSpansFromBadBatchesView := &view.View{
		Name:        StatBadBatchDroppedSpanCount.Name(),
		Measure:     StatBadBatchDroppedSpanCount,
		Description: "The number of spans dropped from span batches with bad data.",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	return []*view.View{
		receivedBatchesView,
		droppedBatchesView,
		receivedSpansView,
		droppedSpansView,
		droppedBadBatchesView,
		droppedSpansFromBadBatchesView,
	}
}

// ServiceNameForNode gets the service name for a specified node. Used for metrics.
func ServiceNameForNode(node *commonpb.Node) string {
	var serviceName string
	if node == nil {
		serviceName = "<nil-batch-node>"
	} else if node.ServiceInfo == nil {
		serviceName = "<nil-service-info>"
	} else if node.ServiceInfo.Name == "" {
		serviceName = "<empty-service-info-name>"
	} else {
		serviceName = node.ServiceInfo.Name
	}
	return serviceName
}

// StatsTagsForBatch gets the stat tags based on the specified processorName, serviceName, and spanFormat.
func StatsTagsForBatch(processorName, serviceName, spanFormat string) []tag.Mutator {
	statsTags := []tag.Mutator{
		tag.Upsert(TagSourceFormatKey, spanFormat),
		tag.Upsert(TagServiceNameKey, serviceName),
		tag.Upsert(TagProcessorNameKey, processorName),
	}

	return statsTags
}
