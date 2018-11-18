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

package internal

// This file contains helpers that are useful to add observability
// with metrics and tracing using OpenCensus to the various pieces
// of the service.

import (
	"context"

	"google.golang.org/grpc"

	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

var (
	tagKeyReceiverName, _ = tag.NewKey("opencensus_receiver")
	tagKeyExporterName, _ = tag.NewKey("opencensus_exporter")
)

var mReceivedSpans = stats.Int64("oc.io/receiver/received_spans", "Counts the number of spans received by the receiver", "1")

var itemsDistribution = view.Distribution(
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 14, 16, 18, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90,
	100, 150, 200, 250, 300, 450, 500, 600, 700, 800, 900, 1000, 1200, 1400, 1600, 1800, 2000,
)

// ViewReceivedSpansReceiver defines the view for the received spans metric.
var ViewReceivedSpansReceiver = &view.View{
	Name:        "oc.io/receiver/received_spans",
	Description: "The number of spans received by the receiver",
	Measure:     mReceivedSpans,
	Aggregation: itemsDistribution,
	TagKeys:     []tag.Key{tagKeyReceiverName},
}

var mExportedSpans = stats.Int64("oc.io/receiver/exported_spans", "Counts the number of exported spans", "1")

// ViewExportedSpans defines the view for exported spans metric.
var ViewExportedSpans = &view.View{
	Name:        "oc.io/receiver/exported_spans",
	Description: "Tracks the number of exported spans",
	Measure:     mExportedSpans,
	Aggregation: itemsDistribution,
	TagKeys:     []tag.Key{tagKeyExporterName},
}

// AllViews has the views for the metrics provided by the agent.
var AllViews = []*view.View{
	ViewReceivedSpansReceiver,
	ViewExportedSpans,
}

// ContextWithReceiverName adds the tag "opencensus_receiver" and the name of the
// receiver as the value, and returns the newly created context.
func ContextWithReceiverName(ctx context.Context, receiverName string) context.Context {
	ctx, _ = tag.New(ctx, tag.Upsert(tagKeyReceiverName, receiverName))
	return ctx
}

// NewReceivedSpansRecorderStreaming creates a function that uses a context created
// from the name of the receiver to record the number of the spans received
// by the receiver.
func NewReceivedSpansRecorderStreaming(lifetimeCtx context.Context, receiverName string) func(*commonpb.Node, []*tracepb.Span) {
	// We create and reuse this context because for streaming RPCs e.g. with gRPC
	// the context doesn't change, so it is more useful for avoid expensively adding
	// keys on each invocation. We can create the context once and then reuse it
	// when recording measurements.
	ctx := ContextWithReceiverName(lifetimeCtx, receiverName)

	return func(ni *commonpb.Node, spans []*tracepb.Span) {
		// TODO: (@odeke-em) perhaps also record information from the node?
		stats.Record(ctx, mReceivedSpans.M(int64(len(spans))))
	}
}

// NewExportedSpansRecorder creates a helper function that'll add the name of the
// creating exporter as a tag value in the context that will be used to count the
// the number of spans exported.
func NewExportedSpansRecorder(exporterName string) func(context.Context, *commonpb.Node, []*tracepb.Span) {
	return func(ctx context.Context, ni *commonpb.Node, spans []*tracepb.Span) {
		ctx, _ = tag.New(ctx, tag.Upsert(tagKeyExporterName, exporterName))
		stats.Record(ctx, mExportedSpans.M(int64(len(spans))))
	}
}

// GRPCServerWithObservabilityEnabled creates a gRPC server that at a bare minimum has
// the OpenCensus ocgrpc server stats handler enabled for tracing and stats.
// Use it instead of invoking grpc.NewServer directly.
func GRPCServerWithObservabilityEnabled(extraOpts ...grpc.ServerOption) *grpc.Server {
	opts := append(extraOpts, grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	return grpc.NewServer(opts...)
}

// SetParentLink tries to retrieve a span from sideCtx and if one exists
// sets its SpanID, TraceID as a link in the span provided. It returns
// true only if it retrieved a parent span from the context.
func SetParentLink(sideCtx context.Context, span *trace.Span) bool {
	parentSpanFromRPC := trace.FromContext(sideCtx)
	if parentSpanFromRPC == nil {
		return false
	}

	psc := parentSpanFromRPC.SpanContext()
	span.AddLink(trace.Link{
		SpanID:  psc.SpanID,
		TraceID: psc.TraceID,
		Type:    trace.LinkTypeParent,
	})
	return true
}
