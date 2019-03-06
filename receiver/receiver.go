// Copyright 2018, OpenCensus Authors
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

	"github.com/census-instrumentation/opencensus-service/consumer"
	_ "github.com/census-instrumentation/opencensus-service/internal/compression/grpc" // load in supported grpc compression encodings
)

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
	StartTraceReception(ctx context.Context, nextConsumer consumer.TraceConsumer) error

	// StopTraceReception tells the receiver that should stop reception,
	// giving it a chance to perform any necessary clean-up.
	StopTraceReception(ctx context.Context) error
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
	StartMetricsReception(ctx context.Context, nextConsumer consumer.MetricsConsumer) error

	// StopMetricsReception tells the receiver that should stop reception,
	// giving it a chance to perform any necessary clean-up.
	StopMetricsReception(ctx context.Context) error
}
