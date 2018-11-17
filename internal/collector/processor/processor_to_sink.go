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

package processor

import (
	"context"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/spansink"
)

type protoProcessorSink struct {
	sourceFormat   string
	protoProcessor SpanProcessor
}

var _ (spansink.Sink) = (*protoProcessorSink)(nil)

// WrapWithSpanSink wraps a processor to be used as a span sink by receivers.
func WrapWithSpanSink(format string, p SpanProcessor) spansink.Sink {
	return &protoProcessorSink{
		sourceFormat:   format,
		protoProcessor: p,
	}
}

func (ps *protoProcessorSink) ReceiveSpans(ctx context.Context, node *commonpb.Node, spans ...*tracepb.Span) (*spansink.Acknowledgement, error) {
	batch := &agenttracepb.ExportTraceServiceRequest{
		Node:  node,
		Spans: spans,
	}

	failures, err := ps.protoProcessor.ProcessSpans(batch, ps.sourceFormat)

	ack := &spansink.Acknowledgement{
		SavedSpans:   uint64(len(batch.Spans)) - failures,
		DroppedSpans: failures,
	}

	return ack, err
}
