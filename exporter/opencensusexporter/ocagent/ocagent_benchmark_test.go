// Copyright 2019, OpenCensus Authors
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

package ocagent

import (
	"encoding/binary"
	"log"
	"sync"
	"testing"
	"time"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

func Benchmark_NoBackend_WithDefaults(b *testing.B) {
	const numExporters = 2
	exportersChan := make(chan *Exporter, numExporters)
	for i := 0; i < cap(exportersChan); i++ {
		exp, err := NewExporter(
			WithInsecure(),
			WithAddress("127.0.0.1:56565"),
		)
		if err != nil {
			log.Fatalf("Failed to create the agent exporter: %v", err)
		}
		defer exp.Stop()
		exportersChan <- exp
	}

	batchedSpans := make([]*tracepb.Span, 0, 10)
	name := &tracepb.TruncatableString{Value: "Benchmark"}
	for i := 0; i < cap(batchedSpans); i++ {
		// In principle there is no translation and this should not make a difference for
		// this test, but, for completeness lets put some data on the spans.
		span := &tracepb.Span{
			Name:      name,
			TraceId:   generateTraceID(uint64(i)),
			SpanId:    generateSpanID(uint64(i)),
			StartTime: timeToTimestamp(time.Now()),
			EndTime:   timeToTimestamp(time.Now().Add(time.Millisecond)),
			Kind:      tracepb.Span_CLIENT,
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"load_generator.span_seq_num":  {Value: &tracepb.AttributeValue_IntValue{IntValue: int64(i)}},
					"load_generator.trace_seq_num": {Value: &tracepb.AttributeValue_IntValue{IntValue: int64(i)}},
				},
			},
		}
		batchedSpans = append(batchedSpans, span)
	}
	batch := &agenttracepb.ExportTraceServiceRequest{
		Node:  NodeWithStartTime("Benchmark-NoBackend"),
		Spans: batchedSpans,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		wg := sync.WaitGroup{}
		for j := 0; j < 10000; j++ {
			wg.Add(1)
			go func() {
				exp := <-exportersChan
				exp.ExportTraceServiceRequest(batch)
				exportersChan <- exp
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func generateTraceID(id uint64) []byte {
	traceID := make([]byte, 16, 16)
	binary.PutUvarint(traceID[:], id)
	return traceID
}

func generateSpanID(id uint64) []byte {
	spanID := make([]byte, 8, 8)
	binary.PutUvarint(spanID[:], id)
	return spanID
}
