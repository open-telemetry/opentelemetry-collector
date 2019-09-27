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

package receiver

import (
	"context"

	_ "github.com/open-telemetry/opentelemetry-collector/compression/grpc" // load in supported grpc compression encodings
)

// Host represents the entity where the receiver is being hosted. It is used to
// allow communication between the receiver and its host.
type Host interface {
	// Context returns a context provided by the host to be used on the receiver
	// operations.
	Context() context.Context

	// ReportFatalError is used to report to the host that the receiver encountered
	// a fatal error (i.e.: an error that the instance can't recover from) after
	// its start function has already returned.
	ReportFatalError(err error)
}

// A TraceReceiver is an "arbitrary data"-to-"trace proto span" converter.
// Its purpose is to translate data from the wild into trace proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// TraceReceiver feeds a consumer.TraceConsumer with data.
//
// For example it could be Zipkin data source which translates
// Zipkin spans into *tracepb.Span-s.
type TraceReceiver interface {
	// TraceSource returns the name of the trace data source.
	TraceSource() string

	// StartTraceReception tells the receiver to start its processing.
	// By convention the consumer of the data received is set at creation time.
	StartTraceReception(host Host) error

	// StopTraceReception tells the receiver that should stop reception,
	// giving it a chance to perform any necessary clean-up.
	StopTraceReception() error
}

// A MetricsReceiver is an "arbitrary data"-to-"metric proto" converter.
// Its purpose is to translate data from the wild into metric proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// MetricsReceiver feeds a consumer.MetricsConsumer with data.
//
// For example it could be Prometheus data source which translates
// Prometheus metrics into *metricpb.Metric-s.
type MetricsReceiver interface {
	// MetricsSource returns the name of the metrics data source.
	MetricsSource() string

	// StartMetricsReception tells the receiver to start its processing.
	// By convention the consumer of the data received is set at creation time.
	StartMetricsReception(host Host) error

	// StopMetricsReception tells the receiver that should stop reception,
	// giving it a chance to perform any necessary clean-up.
	StopMetricsReception() error
}
