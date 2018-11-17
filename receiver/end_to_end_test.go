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

package receiver_test

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/receiver"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus"
)

func Example_endToEnd() {
	// This is what the cmd/ocagent code would look like this.
	// A trace receiver as per the trace receiver
	// configs that have been parsed.
	tr, err := opencensus.New("localhost:55678")
	if err != nil {
		log.Fatalf("Failed to create trace receiver: %v", err)
	}

	// The agent will combine all trace receivers like this.
	trl := []receiver.TraceReceiver{tr}

	// Once we have the span receiver which will connect to the
	// various exporter pipeline i.e. *tracepb.Span->OpenCensus.SpanData
	lsr := new(logSpanSink)
	for _, tr := range trl {
		if err := tr.StartTraceReception(context.Background(), lsr); err != nil {
			log.Fatalf("Failed to start trace receiver: %v", err)
		}
	}

	// Before exiting, stop all the trace receivers
	defer func() {
		for _, tr := range trl {
			_ = tr.StopTraceReception(context.Background())
		}
	}()
	log.Println("Done starting the trace receiver")
	// We are done with the agent-core

	// Now this code would exist in the client application e.g. client code.
	// Create the agent exporter
	oce, err := ocagent.NewExporter(ocagent.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to create ocagent exporter: %v", err)
	}
	defer oce.Stop()

	// Register it as a trace exporter
	trace.RegisterExporter(oce)
	// For demo purposes we are always sampling
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	log.Println("Starting loop")
	ctx, span := trace.StartSpan(context.Background(), "ClientLibrarySpan")
	for i := 0; i < 10; i++ {
		_, span := trace.StartSpan(ctx, "ChildSpan")
		span.Annotatef([]trace.Attribute{
			trace.StringAttribute("type", "Child"),
			trace.Int64Attribute("i", int64(i)),
		}, "This is an annotation")
		<-time.After(100 * time.Millisecond)
		span.End()
		oce.Flush()
	}
	span.End()

	<-time.After(400 * time.Millisecond)
	oce.Flush()
	<-time.After(5 * time.Second)
}

type logSpanSink int

var _ receiver.TraceReceiverSink = (*logSpanSink)(nil)

func (lsr *logSpanSink) ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*receiver.TraceReceiverAcknowledgement, error) {
	spansBlob, _ := json.MarshalIndent(spans, " ", "  ")
	log.Printf("\n****\nNode: %#v\nSpans: %s\n****\n", node, spansBlob)

	return &receiver.TraceReceiverAcknowledgement{SavedSpans: uint64(len(spans))}, nil
}
