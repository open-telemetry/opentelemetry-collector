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
	"github.com/census-instrumentation/opencensus-service/spanreceiver"
	tracetranslator "github.com/census-instrumentation/opencensus-service/translator/trace"
)

type TraceExporter interface {
	ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error
}

type toOCExportersTransformer struct {
	ocTraceExporters []trace.Exporter
}

type TraceExporterSpanReceiver interface {
	TraceExporter
	spanreceiver.SpanReceiver
}

// OCExportersToTraceExporterSpanReceiver is a convenience function that transforms
// traditional OpenCensus trace.Exporter-s into an opencensus-service TraceExporter.
// The resulting TraceExporter ignores the passed in node. To make use of the node,
// please create a custom TraceExporter.
func OCExportersToTraceExporter(ocTraceExporters ...trace.Exporter) TraceExporterSpanReceiver {
	return &toOCExportersTransformer{ocTraceExporters: ocTraceExporters}
}

var _ TraceExporter = (*toOCExportersTransformer)(nil)
var _ spanreceiver.SpanReceiver = (*toOCExportersTransformer)(nil)

func (tse *toOCExportersTransformer) ExportSpanData(ctx context.Context, node *commonpb.Node, spanData ...*trace.SpanData) error {
	for _, sd := range spanData {
		for _, traceExp := range tse.ocTraceExporters {
			traceExp.ExportSpan(sd)
		}
	}
	return nil
}

func (tse *toOCExportersTransformer) ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*spanreceiver.Acknowledgement, error) {
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
	ack := &spanreceiver.Acknowledgement{
		SavedSpans:   uint64(nSaved),
		DroppedSpans: uint64(len(spans) - nSaved),
	}
	return ack, err
}
