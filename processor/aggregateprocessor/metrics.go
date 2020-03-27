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

package aggregateprocessor

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"

	"github.com/open-telemetry/opentelemetry-collector/internal/collector/telemetry"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
)

// Variables related to metrics specific to tail sampling.
var (
	statCountSpansReceived  = stats.Int64("agg_proc_count_spans_received", "Count of spans received by this processor", stats.UnitDimensionless)
	statCountSpansForwarded = stats.Int64("count_spans_forwarded", "Count of spans that were forwarded to a collector peer", stats.UnitDimensionless)
)

// MetricViews return the metrics views according to given telemetry level.
func MetricViews(level telemetry.Level) []*view.View {

	countSpansRcvdView := &view.View{
		Name:        statCountSpansReceived.Name(),
		Measure:     statCountSpansReceived,
		Description: statCountSpansReceived.Description(),
		Aggregation: view.Sum(),
	}

	countSpansFwdView := &view.View{
		Name:        statCountSpansForwarded.Name(),
		Measure:     statCountSpansForwarded,
		Description: statCountSpansForwarded.Description(),
		Aggregation: view.Sum(),
	}

	legacyViews := []*view.View{
		countSpansRcvdView,
		countSpansFwdView,
	}

	return obsreport.ProcessorMetricViews(typeStr, legacyViews)
}
