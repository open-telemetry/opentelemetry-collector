// Copyright 2019, OpenCensus Authors
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
	"context"

	"github.com/census-instrumentation/opencensus-service/data"
)

// MetricsDataProcessor is an interface that receives data.MetricsData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ProcessMetricsData receives data.MetricsData for processing by the MetricsDataProcessor.
type MetricsDataProcessor interface {
	ProcessMetricsData(ctx context.Context, md data.MetricsData) error
}

// TraceDataProcessor is an interface that receives data.TraceData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ProcessTraceData receives data.TraceData for processing by the TraceDataProcessor.
type TraceDataProcessor interface {
	ProcessTraceData(ctx context.Context, td data.TraceData) error
}
