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

	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/spansink"
	tracetranslator "github.com/census-instrumentation/opencensus-service/translator/trace"
)

// TraceExporter is a interface that receives OpenCensus data, converts it as needed, and
// sends it to different destinations.
//
// ExportSpanData receives OpenCensus data for processing by the exporter.
type TraceExporter interface {
	ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error
}

type toOCExportersTransformer struct {
	ocTraceExporters []trace.Exporter
}

// TraceExporterSink is a interface connecting a spansink.Sink and
// an exporter.TraceExporter. The receiver gets data in different serialization formats,
// transforms it to OpenCensus in memory data and sends it to the exporter.
type TraceExporterSink interface {
	TraceExporter
	spansink.Sink
}

// OCExportersToTraceExporter is a convenience function that transforms
// traditional OpenCensus trace.Exporter-s into an opencensus-service TraceExporter.
// The resulting TraceExporter ignores the passed in node. To make use of the node,
// please create a custom TraceExporter.
func OCExportersToTraceExporter(ocTraceExporters ...trace.Exporter) TraceExporterSink {
	return &toOCExportersTransformer{ocTraceExporters: ocTraceExporters}
}

var _ TraceExporter = (*toOCExportersTransformer)(nil)
var _ spansink.Sink = (*toOCExportersTransformer)(nil)

func (tse *toOCExportersTransformer) ExportSpanData(ctx context.Context, node *commonpb.Node, spanData ...*trace.SpanData) error {
	for _, sd := range spanData {
		for _, traceExp := range tse.ocTraceExporters {
			traceExp.ExportSpan(sd)
		}
	}
	return nil
}

func (tse *toOCExportersTransformer) ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*spansink.Acknowledgement, error) {
	// Firstly transform them into spanData.
	spanDataList := make([]*trace.SpanData, 0, len(spans))
	for _, span := range spans {
		spanData, err := tracetranslator.ProtoSpanToOCSpanData(span)
		if err == nil {
			spanDataList = append(spanDataList, spanData)
		}
	}

	// Note: We may want to enhance the SpanData in spanDataList
	// before they get sent to the various exporters.

	// Now invoke ExportSpanData.
	err := tse.ExportSpanData(ctx, node, spanDataList...)
	nSaved := len(spanDataList)
	ack := &spansink.Acknowledgement{
		SavedSpans:   uint64(nSaved),
		DroppedSpans: uint64(len(spans) - nSaved),
	}
	return ack, err
}

// MultiTraceExporters wraps multiple trace exporters in a single one.
func MultiTraceExporters(tes ...TraceExporter) TraceExporterSink {
	return traceExporters(tes)
}

type traceExporters []TraceExporter

// ExportSpanData exports the span data to all trace exporters wrapped by the current one.
func (tes traceExporters) ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error {
	for _, te := range tes {
		_ = te.ExportSpanData(ctx, node, spandata...)
	}
	return nil
}

// ReceiveSpans receives the span data in the protobuf format, translates it, and forwards the transformed
// span data to all trace exporters wrapped by the current one.
func (tes traceExporters) ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*spansink.Acknowledgement, error) {
	spanDataList := make([]*trace.SpanData, 0, len(spans))
	for _, span := range spans {
		spanData, _ := tracetranslator.ProtoSpanToOCSpanData(span)
		if spanData != nil {
			spanDataList = append(spanDataList, spanData)
		}
	}

	err := tes.ExportSpanData(ctx, node, spanDataList...)
	nSaved := len(spanDataList)
	ack := &spansink.Acknowledgement{
		SavedSpans:   uint64(nSaved),
		DroppedSpans: uint64(len(spans) - nSaved),
	}
	return ack, err
}
