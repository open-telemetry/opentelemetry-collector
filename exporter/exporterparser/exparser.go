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

// Package exporterparser provides support for parsing and creating the
// respective exporters given a viper configuration.
// For now it currently only provides statically imported OpenCensus
// exporters like:
//  * Stackdriver Tracing and Monitoring
//  * DataDog
//  * Zipkin
package exporterparser

import (
	"context"

	"go.opencensus.io/trace"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal"
	tracetranslator "github.com/census-instrumentation/opencensus-service/translator/trace"
)

// OcProtoSpansToOCSpanDataInstrumented converts
// OpenCensus Proto TraceData to OpenCensus-Go SpanData.
// The "Instrumented" suffix serves to document that this
// function is traced but also has stats for self-observability.
//
// This is a bootstrapping mechanism for us to re-use as many of
// the OpenCensus-Go trace.SpanData exporters which were written
// by various vendors and contributors. Eventually the goal is to
// get those exporters converted to directly receive
// OpenCensus Proto TraceData.
func OcProtoSpansToOCSpanDataInstrumented(ctx context.Context, exporterName string, te trace.Exporter, td data.TraceData) (aerr error) {
	ctx, span := trace.StartSpan(ctx,
		"opencensus.service.exporter."+exporterName+".ExportTrace",
		trace.WithSampler(trace.NeverSample()))

	span.Annotate([]trace.Attribute{
		trace.Int64Attribute("n_spans", int64(len(td.Spans))),
	}, "")

	defer func() {
		if aerr != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeInternal, Message: aerr.Error()})
		}
		span.End()
	}()

	var errs []error
	var goodSpans []*tracepb.Span
	for _, span := range td.Spans {
		sd, err := tracetranslator.ProtoSpanToOCSpanData(span)
		if err == nil {
			te.ExportSpan(sd)
			goodSpans = append(goodSpans, span)
		} else {
			errs = append(errs, err)
		}
	}

	return internal.CombineErrors(errs)
}
