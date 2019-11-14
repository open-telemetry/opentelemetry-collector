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
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

var (
	statBatchSize               = stats.Int64("batch_size", "Size of batches sent from the batcher (in span)", stats.UnitDimensionless)
	statNodesAddedToBatches     = stats.Int64("nodes_added_to_batches", "Count of nodes that are being batched.", stats.UnitDimensionless)
	statNodesRemovedFromBatches = stats.Int64("nodes_removed_from_batches", "Number of nodes that have been removed from batching.", stats.UnitDimensionless)

	statBatchSizeTriggerSend = stats.Int64("batch_size_trigger_send", "Number of times the batch was sent due to a size trigger", stats.UnitDimensionless)
	statTimeoutTriggerSend   = stats.Int64("timeout_trigger_send", "Number of times the batch was sent due to a timeout trigger", stats.UnitDimensionless)
	statBatchOnDeadNode      = stats.Int64("removed_node_send", "Number of times the batch was sent due to spans being added for a no longer active node", stats.UnitDimensionless)
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

	batchSizeAggregation := view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000, 50000, 100000)

	batchSizeView := &view.View{
		Name:        statBatchSize.Name(),
		Measure:     statBatchSize,
		Description: statBatchSize.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: batchSizeAggregation,
	}

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

	countBatchOnDeadNode := &view.View{
		Name:        statBatchOnDeadNode.Name(),
		Measure:     statBatchOnDeadNode,
		Description: statBatchOnDeadNode.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	return []*view.View{
		batchSizeView,
		nodesAddedToBatchesView,
		nodesRemovedFromBatchesView,
		countBatchSizeTriggerSendView,
		countTimeoutTriggerSendView,
		countBatchOnDeadNode,
	}
}
