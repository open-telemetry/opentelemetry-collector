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

package kafkareceiver

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	tagInstanceName, _ = tag.NewKey("name")

	statMessageCount     = stats.Int64("kafka_receiver_messages", "Number of received messages", stats.UnitDimensionless)
	statMessageOffset    = stats.Int64("kafka_receiver_current_offset", "Current message offset", stats.UnitDimensionless)
	statMessageOffsetLag = stats.Int64("kafka_receiver_offset_lag", "Current offset lag", stats.UnitDimensionless)

	statPartitionStart = stats.Int64("kafka_receiver_partition_start", "Number of started partitions", stats.UnitDimensionless)
	statPartitionClose = stats.Int64("kafka_receiver_partition_close", "Number of finished partitions", stats.UnitDimensionless)
)

// MetricViews return metric views for Kafka receiver.
func MetricViews() []*view.View {
	tagKeys := []tag.Key{tagInstanceName}

	countMessages := &view.View{
		Name:        statMessageCount.Name(),
		Measure:     statMessageCount,
		Description: statMessageCount.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	lastValueOffset := &view.View{
		Name:        statMessageOffset.Name(),
		Measure:     statMessageOffset,
		Description: statMessageOffset.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.LastValue(),
	}

	lastValueOffsetLag := &view.View{
		Name:        statMessageOffsetLag.Name(),
		Measure:     statMessageOffsetLag,
		Description: statMessageOffsetLag.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.LastValue(),
	}

	countPartitionStart := &view.View{
		Name:        statPartitionStart.Name(),
		Measure:     statPartitionStart,
		Description: statPartitionStart.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	countPartitionClose := &view.View{
		Name:        statPartitionClose.Name(),
		Measure:     statPartitionClose,
		Description: statPartitionClose.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	return []*view.View{
		countMessages,
		lastValueOffset,
		lastValueOffsetLag,
		countPartitionStart,
		countPartitionClose,
	}
}
