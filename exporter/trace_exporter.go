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

package exporter

import (
	"context"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

// TraceExporter is a interface that receives OpenCensus data, converts it as needed, and
// sends it to different destinations.
//
// ExportSpans receives OpenCensus proto spans for processing by the exporter.
type TraceExporter interface {
	ExportSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) error
}

// TraceExporterSink is a interface connecting a TraceReceiverSink and
// an exporter.TraceExporter. The receiver gets data in different serialization formats,
// transforms it to OpenCensus in memory data and sends it to the exporter.
type TraceExporterSink interface {
	TraceExporter
	receiver.TraceReceiverSink
}

// MultiTraceExporters wraps multiple trace exporters in a single one.
func MultiTraceExporters(tes ...TraceExporter) TraceExporterSink {
	return traceExporters(tes)
}

type traceExporters []TraceExporter

// ExportSpans exports the span data to all trace exporters wrapped by the current one.
func (tes traceExporters) ExportSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) error {
	for _, te := range tes {
		_ = te.ExportSpans(ctx, node, spans...)
	}
	return nil
}

// ReceiveSpans receives the span data in the protobuf format, translates it, and forwards the transformed
// span data to all trace exporters wrapped by the current one.
func (tes traceExporters) ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*receiver.TraceReceiverAcknowledgement, error) {
	for _, te := range tes {
		_ = te.ExportSpans(ctx, node, spans...)
	}

	ack := &receiver.TraceReceiverAcknowledgement{
		SavedSpans: uint64(len(spans)),
	}
	return ack, nil
}
