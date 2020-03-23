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

package batchprocessor

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/open-telemetry/opentelemetry-collector/internal/collector/telemetry"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

var (
	statNodesAddedToBatches     = stats.Int64("nodes_added_to_batches", "Count of nodes that are being batched.", stats.UnitDimensionless)
	statNodesRemovedFromBatches = stats.Int64("nodes_removed_from_batches", "Number of nodes that have been removed from batching.", stats.UnitDimensionless)

	statBatchSizeTriggerSend = stats.Int64("batch_size_trigger_send", "Number of times the batch was sent due to a size trigger", stats.UnitDimensionless)
	statTimeoutTriggerSend   = stats.Int64("timeout_trigger_send", "Number of times the batch was sent due to a timeout trigger", stats.UnitDimensionless)
)

// MetricViews returns the metrics views related to batching
func MetricViews(level telemetry.Level) []*view.View {
	if level == telemetry.None {
		return nil
	}

	tagKeys := processor.MetricTagKeys(level)
	if tagKeys == nil {
		return nil
	}

	processorTagKeys := []tag.Key{processor.TagProcessorNameKey}

	nodesAddedToBatchesView := &view.View{
		Name:        statNodesAddedToBatches.Name(),
		Measure:     statNodesAddedToBatches,
		Description: statNodesAddedToBatches.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	nodesRemovedFromBatchesView := &view.View{
		Name:        statNodesRemovedFromBatches.Name(),
		Measure:     statNodesRemovedFromBatches,
		Description: statNodesRemovedFromBatches.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	countBatchSizeTriggerSendView := &view.View{
		Name:        statBatchSizeTriggerSend.Name(),
		Measure:     statBatchSizeTriggerSend,
		Description: statBatchSizeTriggerSend.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	countTimeoutTriggerSendView := &view.View{
		Name:        statTimeoutTriggerSend.Name(),
		Measure:     statTimeoutTriggerSend,
		Description: statTimeoutTriggerSend.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	legacyViews := []*view.View{
		nodesAddedToBatchesView,
		nodesRemovedFromBatchesView,
		countBatchSizeTriggerSendView,
		countTimeoutTriggerSendView,
	}

	return obsreport.ProcessorMetricViews(typeStr, legacyViews)
}
