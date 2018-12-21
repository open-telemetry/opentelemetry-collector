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

package testhelper

import (
	"context"
	"encoding/json"
	"sync"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

// ConcurrentSpanSink acts as a trace receiver for use in tests.
type ConcurrentSpanSink struct {
	mu     sync.Mutex
	traces []*agenttracepb.ExportTraceServiceRequest
}

var _ receiver.TraceReceiverSink = (*ConcurrentSpanSink)(nil)

// ReceiveTraceData stores traces in the span sink for verification in tests.
func (css *ConcurrentSpanSink) ReceiveTraceData(ctx context.Context, td data.TraceData) (*receiver.TraceReceiverAcknowledgement, error) {
	css.mu.Lock()
	defer css.mu.Unlock()

	css.traces = append(css.traces, &agenttracepb.ExportTraceServiceRequest{
		Node:  td.Node,
		Spans: td.Spans,
	})

	ack := &receiver.TraceReceiverAcknowledgement{
		SavedSpans: uint64(len(td.Spans)),
	}

	return ack, nil
}

// AllTraces returns the traces sent to the test sink.
func (css *ConcurrentSpanSink) AllTraces() []*agenttracepb.ExportTraceServiceRequest {
	css.mu.Lock()
	defer css.mu.Unlock()

	return css.traces[:]
}

// ToJSON marshals a generic interface to JSON to enable easy comparisons.
func ToJSON(v interface{}) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
