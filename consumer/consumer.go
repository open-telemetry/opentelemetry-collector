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

// Package consumer contains interfaces that receive and process consumerdata.
package consumer

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

// MetricsConsumer is an interface that receives consumerdata.MetricsData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ConsumeMetricsData receives consumerdata.MetricsData for processing by the MetricsConsumer.
type MetricsConsumer interface {
	ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error
}

// MetricsConsumerV2 is the new metrics consumer interface that receives data.MetricData, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type MetricsConsumerV2 interface {
	ConsumeMetrics(ctx context.Context, md data.MetricData) error
}

// BaseTraceConsumer defines a common interface for TraceConsumer and TraceConsumerV2.
type BaseTraceConsumer interface{}

// TraceConsumer is an interface that receives consumerdata.TraceData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ConsumeTraceData receives consumerdata.TraceData for processing by the TraceConsumer.
type TraceConsumer interface {
	BaseTraceConsumer
	ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error
}

// TraceConsumerV2 is an interface that receives data.TraceData, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type TraceConsumerV2 interface {
	BaseTraceConsumer
	// ConsumeTrace receives data.TraceData for processing.
	ConsumeTrace(ctx context.Context, td data.TraceData) error
}
