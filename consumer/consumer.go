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

	"go.opentelemetry.io/collector/consumer/pdata"
)

// Capabilities describes the capabilities of a Processor.
type Capabilities struct {
	// MutatesData is set to true if Consume* function of the Consumer modifies the input
	// pdata.Traces, pdata.Metrics, or pdata.Logs argument.
	// Consumer which modify the input data MUST set this flag to true. If the Consumer
	// does not modify the data it MUST set this flag to false. If the Consumer creates
	// a copy of the data before modifying then this flag can be safely set to false.
	MutatesData bool
}

type Consumer interface {
	// GetCapabilities must return the capabilities of the Consumer.
	GetCapabilities() Capabilities
}

// TracesConsumer is an interface that receives pdata.Traces, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type TracesConsumer interface {
	Consumer
	// ConsumeTraces receives pdata.Traces for consumption.
	ConsumeTraces(ctx context.Context, td pdata.Traces) error
}

// MetricsConsumer is the new metrics consumer interface that receives pdata.MetricData, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type MetricsConsumer interface {
	Consumer
	// ConsumeMetrics receives pdata.Metrics for consumption.
	ConsumeMetrics(ctx context.Context, md pdata.Metrics) error
}

// LogsConsumer is an interface that receives pdata.Logs, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type LogsConsumer interface {
	Consumer
	// ConsumeLogs receives pdata.Logs for consumption.
	ConsumeLogs(ctx context.Context, ld pdata.Logs) error
}
