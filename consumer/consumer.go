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

package consumer

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/data"
)

// MetricsConsumer is an interface that receives data.MetricsData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ConsumeMetricsData receives data.MetricsData for processing by the MetricsConsumer.
type MetricsConsumer interface {
	ConsumeMetricsData(ctx context.Context, md data.MetricsData) error
}

// TraceConsumer is an interface that receives data.TraceData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ConsumeTraceData receives data.TraceData for processing by the TraceConsumer.
type TraceConsumer interface {
	ConsumeTraceData(ctx context.Context, td data.TraceData) error
}

// DataConsumer is a union type that can accept traces and/or metrics.
type DataConsumer interface {
	TraceConsumer
	MetricsConsumer
}
