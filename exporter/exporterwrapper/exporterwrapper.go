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

// Package exporterwrapper provides support for wrapping OC go library trace.Exporter into a
// consumer.TraceConsumer.
// For now it currently only provides statically imported OpenCensus
// exporters like:
//  * Zipkin
package exporterwrapper

import (
	"context"

	"go.opencensus.io/trace"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/open-telemetry/opentelemetry-service/data"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterhelper"
	"github.com/open-telemetry/opentelemetry-service/internal"
	spandatatranslator "github.com/open-telemetry/opentelemetry-service/translator/trace/spandata"
)

// NewExporterWrapper returns a consumer.TraceConsumer that converts OpenCensus Proto TraceData
// to OpenCensus-Go SpanData and calls into the given trace.Exporter.
//
// This is a bootstrapping mechanism for us to re-use as many of
// the OpenCensus-Go trace.SpanData exporters which were written
// by various vendors and contributors. Eventually the goal is to
// get those exporters converted to directly receive
// OpenCensus Proto TraceData.
func NewExporterWrapper(exporterName string, spanName string, ocExporter trace.Exporter) (exporter.TraceExporter, error) {
	return exporterhelper.NewTraceExporter(
		exporterName,
		func(ctx context.Context, td data.TraceData) (int, error) {
			return PushOcProtoSpansToOCTraceExporter(ocExporter, td)
		},
		exporterhelper.WithSpanName(spanName),
		exporterhelper.WithRecordMetrics(true),
	)
}

// TODO: Remove PushOcProtoSpansToOCTraceExporter after aws-xray is changed to ExporterWrapper.

// PushOcProtoSpansToOCTraceExporter pushes TraceData to the given trace.Exporter by converting the
// protos to trace.SpanData.
func PushOcProtoSpansToOCTraceExporter(ocExporter trace.Exporter, td data.TraceData) (int, error) {
	var errs []error
	var goodSpans []*tracepb.Span
	for _, span := range td.Spans {
		sd, err := spandatatranslator.ProtoSpanToOCSpanData(span)
		if err == nil {
			ocExporter.ExportSpan(sd)
			goodSpans = append(goodSpans, span)
		} else {
			errs = append(errs, err)
		}
	}

	return len(td.Spans) - len(goodSpans), internal.CombineErrors(errs)
}
