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

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

// TraceExporter is a interface that receives OpenCensus data, converts it as needed, and
// sends it to different destinations.
//
// ExportSpans receives OpenCensus data.TraceData for processing by the exporter.
type TraceExporter interface {
	ExportSpans(ctx context.Context, td data.TraceData) error
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
func (tes traceExporters) ExportSpans(ctx context.Context, td data.TraceData) error {
	for _, te := range tes {
		_ = te.ExportSpans(ctx, td)
	}
	return nil
}

// ReceiveTraceData receives the span data in the protobuf format, translates it, and forwards the transformed
// span data to all trace exporters wrapped by the current one.
func (tes traceExporters) ReceiveTraceData(ctx context.Context, td data.TraceData) (*receiver.TraceReceiverAcknowledgement, error) {
	for _, te := range tes {
		_ = te.ExportSpans(ctx, td)
	}

	ack := &receiver.TraceReceiverAcknowledgement{
		SavedSpans: uint64(len(td.Spans)),
	}
	return ack, nil
}
