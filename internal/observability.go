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

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

var tagKeyInterceptorName, _ = tag.NewKey("opencensus_interceptor")
var mReceivedSpans = stats.Int64("oc.io/interceptor/received_spans", "Counts the number of spans received by the interceptor", "1")

var ViewReceivedSpansInterceptor = &view.View{
	Name:        "oc.io/interceptor/received_spans",
	Description: "The number of spans received by the interceptor",
	Measure:     mReceivedSpans,
	Aggregation: view.Distribution(
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 14, 16, 18, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90,
		100, 150, 200, 250, 300, 450, 500, 600, 700, 800, 900, 1000, 1200, 1400, 1600, 1800, 2000,
	),
	TagKeys: []tag.Key{tagKeyInterceptorName},
}

var AllViews = []*view.View{
	ViewReceivedSpansInterceptor,
}

// NewReceivedSpansRecorderStreaming creates a function that uses a context created
// from the name of the interceptor to record the number of the spans received
// by the interceptor.
func NewReceivedSpansRecorderStreaming(lifetimeCtx context.Context, interceptorName string) func(*commonpb.Node, []*tracepb.Span) {
	// We create and reuse this context because for streaming RPCs e.g. with gRPC
	// the context doesn't change, so it is more useful for avoid expensively adding
	// keys on each invocation. We can create the context once and then reuse it
	// when recording measurements.
	ctx, _ := tag.New(lifetimeCtx, tag.Upsert(tagKeyInterceptorName, interceptorName))

	return func(ni *commonpb.Node, spans []*tracepb.Span) {
		// TODO: (@odeke-em) perhaps also record information from the node?
		stats.Record(ctx, mReceivedSpans.M(int64(len(spans))))
	}
}

// GRPCServerWithObservabilityEnabled creates a gRPC server that at a bare minimum has
// the OpenCensus ocgrpc server stats handler enabled for tracing and stats.
// Use it instead of invoking grpc.NewServer directly.
func GRPCServerWithObservabilityEnabled(extraOpts ...grpc.ServerOption) *grpc.Server {
	opts := append(extraOpts, grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	return grpc.NewServer(opts...)
}
