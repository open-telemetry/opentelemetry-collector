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

package internal

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var tagInstance, _ = tag.NewKey("instance")

var statUpStatus = stats.Int64("up", "Whether the endpoint is alive or not", stats.UnitDimensionless)

func MetricViews() []*view.View {
	return []*view.View{
		{
			Name:        statUpStatus.Name(),
			Measure:     statUpStatus,
			Description: statUpStatus.Description(),
			TagKeys:     []tag.Key{tagInstance},
			Aggregation: view.LastValue(),
		},
	}
}

func recordInstanceAsUp(ctx context.Context, instanceValue string) context.Context {
	ctx, _ = tag.New(ctx, tag.Upsert(tagInstance, instanceValue))
	stats.Record(ctx, statUpStatus.M(1))
	return ctx
}

func recordInstanceAsDown(ctx context.Context, instanceValue string) context.Context {
	ctx, _ = tag.New(ctx, tag.Upsert(tagInstance, instanceValue))
	stats.Record(ctx, statUpStatus.M(0))
	return ctx
}
