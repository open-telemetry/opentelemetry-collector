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

	"github.com/census-instrumentation/opencensus-service/internal"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricpb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

// A TraceReceiver is an "arbitrary data"-to-"trace proto span" converter.
// Its purpose is to translate data from the wild into trace proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// TraceReceiver feeds a TraceReceiverSink with data.
//
// For example it could be Zipkin data source which translates
// Zipkin spans into *tracepb.Span-s.
//
// StartTraceReception tells the receiver to start its processing.
//
// StopTraceReception tells the receiver that should stop reception,
// giving it a chance to perform any necessary clean-up.
type TraceReceiver interface {
	StartTraceReception(ctx context.Context, destination TraceReceiverSink) error
	StopTraceReception(ctx context.Context) error
}

// TraceReceiverSink is an interface that receives spans from a Node identifier.
type TraceReceiverSink interface {
	ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*TraceReceiverAcknowledgement, error)
}

// TraceReceiverAcknowledgement struct reports the number of saved and dropped spans in a
// ReceiveSpans call.
type TraceReceiverAcknowledgement struct {
	SavedSpans   uint64
	DroppedSpans uint64
}

// A MetricsReceiver is an "arbitrary data"-to-"metric proto" converter.
// Its purpose is to translate data from the wild into metric proto accompanied
// by a *commonpb.Node to uniquely identify where that data comes from.
// MetricsReceiver feeds a MetricsReceiverSink with data.
//
// For example it could be Prometheus data source which translates
// Prometheus metrics into *metricpb.Metric-s.
type MetricsReceiver interface {
	StartMetricsReception(ctx context.Context, destination MetricsReceiverSink) error
	StopMetricsReception(ctx context.Context) error
}

// MetricsReceiverSink is an interface that receives metrics from a Node identifier.
type MetricsReceiverSink interface {
	ReceiveMetrics(ctx context.Context, node *commonpb.Node, resource *resourcepb.Resource, metrics ...*metricpb.Metric) (*MetricsReceiverAcknowledgement, error)
}

// MetricsReceiverAcknowledgement struct reports the number of saved and dropped spans in a
// ReceiveSpans call.
type MetricsReceiverAcknowledgement struct {
	SavedMetrics   uint64
	DroppedMetrics uint64
}

// MultiTraceReceiver wraps multiple trace receivers in a single one.
func MultiTraceReceiver(trs ...TraceReceiver) TraceReceiver {
	return traceReceivers(trs)
}

type traceReceivers []TraceReceiver

func (trs traceReceivers) StartTraceReception(ctx context.Context, destination TraceReceiverSink) error {
	var errs []error
	for _, tr := range trs {
		err := tr.StartTraceReception(ctx, destination)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return internal.CombineErrors(errs)
}

func (trs traceReceivers) StopTraceReception(ctx context.Context) error {
	var errs []error
	for _, tr := range trs {
		err := tr.StopTraceReception(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return internal.CombineErrors(errs)
}

// MultiMetricsReceiver wraps multiple metrics receivers in a single one.
func MultiMetricsReceiver(mrs ...MetricsReceiver) MetricsReceiver {
	return metricsReceivers(mrs)
}

type metricsReceivers []MetricsReceiver

func (mrs metricsReceivers) StartMetricsReception(ctx context.Context, destination MetricsReceiverSink) error {
	var errs []error
	for _, mr := range mrs {
		err := mr.StartMetricsReception(ctx, destination)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return internal.CombineErrors(errs)
}

func (mrs metricsReceivers) StopMetricsReception(ctx context.Context) error {
	var errs []error
	for _, mr := range mrs {
		err := mr.StopMetricsReception(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return internal.CombineErrors(errs)
}
