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

package interceptor

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/metricsreceiver"
	"github.com/census-instrumentation/opencensus-service/spanreceiver"
)

// A TraceInterceptor is an "arbitrary data"-to-"trace proto span" converter.
// Its purpose is to translate data from the wild into trace proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// TraceInterceptor feeds a spanreceiver.SpanReceiver with data.
//
// For example it could be Zipkin data source which translates
// Zipkin spans into *tracepb.Span-s.
//
// StartTraceInterception tells the interceptor to start its processing.
//
// StopTraceInterception tells the interceptor that should stop interception,
// giving it a chance to perform any necessary clean-up.
type TraceInterceptor interface {
	StartTraceInterception(ctx context.Context, destination spanreceiver.SpanReceiver) error
	StopTraceInterception(ctx context.Context) error
}

// A MetricsInterceptor is an "arbitrary data"-to-"metric proto" converter.
// Its purpose is to translate data from the wild into metric proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// MetricsInterceptor feeds a metricsreceiver.MetricsReceiver with data.
//
// For example it could be Prometheus data source which translates
// Prometheus metrics into *metricpb.Metric-s.
type MetricsInterceptor interface {
	StartMetricsInterception(ctx context.Context, destination metricsreceiver.MetricsReceiver) error
	StopMetricsInterception(ctx context.Context) error
}
