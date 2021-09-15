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

//go:build enable_unstable
// +build enable_unstable

package internal

import (
	"context"
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	currentlyDispatchedBatches = stats.Int64(
		"/currently_dispatched_batches",
		"Number of batches that are currently being sent",
		stats.UnitDimensionless)

	totalDispatchedBatches = stats.Int64(
		"/total_dispatched_batches",
		"Total number of batches which were processed",
		stats.UnitDimensionless)

	queueNameKey = tag.MustNewKey("queue_name")
)

func recordCurrentlyDispatchedBatches(ctx context.Context, dispatchedBatches int, queueName string) {
	ctx, err := tag.New(ctx, tag.Insert(queueNameKey, queueName))
	if err != nil {
		return
	}

	stats.Record(ctx, currentlyDispatchedBatches.M(int64(dispatchedBatches)))
}

func recordBatchDispatched(ctx context.Context, queueName string) {
	ctx, err := tag.New(ctx, tag.Insert(queueNameKey, queueName))
	if err != nil {
		return
	}

	stats.Record(ctx, totalDispatchedBatches.M(int64(1)))
}

// ExporterHelperInternalViews return the metrics views according to given telemetry level.
func ExporterHelperInternalViews() []*view.View {

	return []*view.View{
		{
			Name:        currentlyDispatchedBatches.Name(),
			Description: currentlyDispatchedBatches.Description(),
			Measure:     currentlyDispatchedBatches,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{queueNameKey},
		},
		{
			Name:        totalDispatchedBatches.Name(),
			Description: totalDispatchedBatches.Description(),
			Measure:     totalDispatchedBatches,
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{queueNameKey},
		},
	}
}

var onceMetrics sync.Once

// RegisterMetrics registers a set of metric views used by the internal package
func RegisterMetrics() {
	onceMetrics.Do(func() {
		_ = view.Register(ExporterHelperInternalViews()...)
	})
}
