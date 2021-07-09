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

package obsreport

import (
	"context"
	"time"

	"go.opencensus.io/stats"

	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

type pipelineStartContextKey struct{}

func recordPipelineStart(ctx context.Context) context.Context {
	return context.WithValue(ctx, pipelineStartContextKey{}, []time.Time{time.Now()})
}

// MergeContexts merges pipeline start times from the second context into the
// first context. It does not preserve any other attributes of the second context.
func MergeContexts(base, merged context.Context) context.Context {
	startTimesA, _ := base.Value(pipelineStartContextKey{}).([]time.Time)
	startTimesB, _ := merged.Value(pipelineStartContextKey{}).([]time.Time)
	return context.WithValue(base, pipelineStartContextKey{}, append(startTimesA, startTimesB...))
}

func recordPipelineDuration(ctx context.Context) {
	startTimes, _ := ctx.Value(pipelineStartContextKey{}).([]time.Time)
	for _, startTime := range startTimes {
		stats.Record(
			ctx,
			obsmetrics.PipelineProcessingDuration.M(time.Since(startTime).Seconds()))
	}
}
