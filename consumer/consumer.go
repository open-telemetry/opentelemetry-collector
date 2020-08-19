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

// Package consumer contains interfaces that receive and process consumerdata.
package consumer

import (
	"context"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// MetricsConsumerBase defines a common interface for MetricsConsumerOld and MetricsConsumer.
type MetricsConsumerBase interface{}

// MetricsConsumerOld is an interface that receives consumerdata.MetricsData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ConsumeMetricsData receives consumerdata.MetricsData for processing by the MetricsConsumer.
type MetricsConsumerOld interface {
	MetricsConsumerBase
	ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error
}

// MetricsConsumer is the new metrics consumer interface that receives pdata.MetricData, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type MetricsConsumer interface {
	MetricsConsumerBase
	ConsumeMetrics(ctx context.Context, md pdata.Metrics) error
}

// TraceConsumerBase defines a common interface for TraceConsumerOld and TraceConsumer.
type TraceConsumerBase interface{}

// TraceConsumerOld is an interface that receives consumerdata.TraceData, process it as needed, and
// sends it to the next processing node if any or to the destination.
//
// ConsumeTraceData receives consumerdata.TraceData for processing by the TraceConsumer.
type TraceConsumerOld interface {
	TraceConsumerBase
	ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error
}

// TraceConsumer is an interface that receives pdata.Traces, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type TraceConsumer interface {
	TraceConsumerBase
	// ConsumeTraces receives pdata.Traces for processing.
	ConsumeTraces(ctx context.Context, td pdata.Traces) error
}

// LogsConsumer is an interface that receives pdata.Logs, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type LogsConsumer interface {
	// ConsumeLogs receives pdata.Logs for processing.
	ConsumeLogs(ctx context.Context, ld pdata.Logs) error
}
