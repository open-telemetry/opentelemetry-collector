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
	"github.com/open-telemetry/opentelemetry-collector/component"
	_ "github.com/open-telemetry/opentelemetry-collector/compression/grpc" // load in supported grpc compression encodings
)

// Receiver defines functions that trace and metric receivers must implement.
type Receiver interface {
	component.Component
}

// A TraceReceiver is an "arbitrary data"-to-"trace proto span" converter.
// Its purpose is to translate data from the wild into trace proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// TraceReceiver feeds a consumer.TraceConsumer with data.
//
// For example it could be Zipkin data source which translates
// Zipkin spans into *tracepb.Span-s.
type TraceReceiver interface {
	Receiver

	// TraceSource returns the name of the trace data source.
	TraceSource() string
}

// A MetricsReceiver is an "arbitrary data"-to-"metric proto" converter.
// Its purpose is to translate data from the wild into metric proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// MetricsReceiver feeds a consumer.MetricsConsumer with data.
//
// For example it could be Prometheus data source which translates
// Prometheus metrics into *metricpb.Metric-s.
type MetricsReceiver interface {
	Receiver

	// MetricsSource returns the name of the metrics data source.
	MetricsSource() string
}
