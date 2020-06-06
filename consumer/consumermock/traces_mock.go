/*
 * Copyright The OpenTelemetry Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consumermock

import (
	"context"
	"sync"

	"go.uber.org/atomic"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var _ consumer.TraceConsumer = (*Trace)(nil)
var _ consumer.TraceConsumerOld = (*Trace)(nil)

type Trace struct {
	sync.Mutex
	consumeTraceError error // to be returned by Consume, if set
	spansReceived     atomic.Uint64
	receivedTraces    []pdata.Traces
	receivedTracesOld []consumerdata.TraceData
}

func (tc *Trace) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	if tc.consumeTraceError != nil {
		return tc.consumeTraceError
	}

	tc.spansReceived.Add(uint64(td.SpanCount()))

	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		ils := rs.At(i).InstrumentationLibrarySpans()
		for j := 0; j < ils.Len(); j++ {
			spans := ils.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				var spanSeqnum int64
				var traceSeqnum int64

				seqnumAttr, ok := span.Attributes().Get("load_generator.span_seq_num")
				if ok {
					spanSeqnum = seqnumAttr.IntVal()
				}

				seqnumAttr, ok = span.Attributes().Get("load_generator.trace_seq_num")
				if ok {
					traceSeqnum = seqnumAttr.IntVal()
				}

				// Ignore the seqnums for now. We will use them later.
				_ = spanSeqnum
				_ = traceSeqnum
			}
		}
	}

	tc.Lock()
	defer tc.Unlock()
	tc.receivedTraces = append(tc.receivedTraces, td)

	return nil
}

// ConsumeTraceData consumes trace data in old representation
func (tc *Trace) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	if tc.consumeTraceError != nil {
		return tc.consumeTraceError
	}

	tc.spansReceived.Add(uint64(len(td.Spans)))

	for _, span := range td.Spans {
		if span == nil {
			continue
		}
		var spanSeqnum int64
		var traceSeqnum int64

		if span.Attributes != nil {
			seqnumAttr, ok := span.Attributes.AttributeMap["load_generator.span_seq_num"]
			if ok {
				spanSeqnum = seqnumAttr.GetIntValue()
			}

			seqnumAttr, ok = span.Attributes.AttributeMap["load_generator.trace_seq_num"]
			if ok {
				traceSeqnum = seqnumAttr.GetIntValue()
			}

			// Ignore the seqnums for now. We will use them later.
			_ = spanSeqnum
			_ = traceSeqnum
		}
	}

	tc.Lock()
	defer tc.Unlock()
	tc.receivedTracesOld = append(tc.receivedTracesOld, td)

	return nil
}

// ClearReceivedItems clears the list of received traces and metrics. Note: counters
// return by DataItemsReceived() are not cleared, they are cumulative.
func (tc *Trace) ClearReceivedItems() {
	tc.Lock()
	defer tc.Unlock()
	tc.receivedTraces = nil
	tc.receivedTracesOld = nil
}

// SpansReceived returns number of spans received by the consumer.
func (tc *Trace) SpansReceived() uint64 {
	return tc.spansReceived.Load()
}

func (tc *Trace) Traces() []pdata.Traces {
	tc.Lock()
	defer tc.Unlock()
	return tc.receivedTraces
}

func (tc *Trace) TracesOld() []consumerdata.TraceData {
	tc.Lock()
	defer tc.Unlock()
	return tc.receivedTracesOld
}

func (tc *Trace) SetConsumeTraceError(err error) {
	tc.consumeTraceError = err
}

