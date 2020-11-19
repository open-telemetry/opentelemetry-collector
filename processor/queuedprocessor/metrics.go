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

package queuedprocessor

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/processor"
)

// Variables related to metrics specific to queued processor.
var (
	statInQueueLatencyMs = stats.Int64("queue_latency", "Latency (in milliseconds) that a batch stayed in queue", stats.UnitMilliseconds)
	statSendLatencyMs    = stats.Int64("send_latency", "Latency (in milliseconds) to send a batch", stats.UnitMilliseconds)
	statSuccessSendOps   = stats.Int64("success_send", "Number of successful send operations", stats.UnitDimensionless)
	statFailedSendOps    = stats.Int64("fail_send", "Number of failed send operations", stats.UnitDimensionless)
	statQueueLength      = stats.Int64("queue_length", "Current length of the queue (in batches)", stats.UnitDimensionless)

	latencyDistributionAggregation = view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000, 50000)

	queueLengthView = &view.View{
		Name:        statQueueLength.Name(),
		Measure:     statQueueLength,
		Description: "Current number of batches in the queue",
		TagKeys:     []tag.Key{processor.TagProcessorNameKey},
		Aggregation: view.LastValue(),
	}
	sendLatencyView = &view.View{
		Name:        statSendLatencyMs.Name(),
		Measure:     statSendLatencyMs,
		Description: "The latency of the successful send operations.",
		TagKeys:     []tag.Key{processor.TagProcessorNameKey},
		Aggregation: latencyDistributionAggregation,
	}
	inQueueLatencyView = &view.View{
		Name:        statInQueueLatencyMs.Name(),
		Measure:     statInQueueLatencyMs,
		Description: "The \"in queue\" latency of the successful send operations.",
		TagKeys:     []tag.Key{processor.TagProcessorNameKey},
		Aggregation: latencyDistributionAggregation,
	}
)

// MetricViews return the metrics views according to given telemetry level.
func MetricViews() []*view.View {
	tagKeys := processor.MetricTagKeys()

	countSuccessSendView := &view.View{
		Name:        statSuccessSendOps.Name(),
		Measure:     statSuccessSendOps,
		Description: "The number of successful send operations performed by queued_retry processor",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}
	countFailuresSendView := &view.View{
		Name:        statFailedSendOps.Name(),
		Measure:     statFailedSendOps,
		Description: "The number of failed send operations performed by queued_retry processor",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	legacyViews := []*view.View{queueLengthView, countSuccessSendView, countFailuresSendView, sendLatencyView, inQueueLatencyView}

	return obsreport.ProcessorMetricViews(typeStr, legacyViews)
}
