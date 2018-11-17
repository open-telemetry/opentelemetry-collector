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
	"go.uber.org/zap"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
)

// SpanProcessor handles batches of spans converted to OpenCensus proto format.
type SpanProcessor interface {
	// ProcessSpans processes spans and return with the number of spans that failed and an error.
	ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error)
	// TODO: (@pjanotti) For shutdown improvement, the interface needs a method to attempt that.
}

// An initial processor that does not sends the data to any destination but helps debugging.
type debugSpanProcessor struct{ logger *zap.Logger }

var _ SpanProcessor = (*debugSpanProcessor)(nil)

func (sp *debugSpanProcessor) ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error) {
	if batch.Node == nil {
		sp.logger.Warn("Received batch with nil Node", zap.String("format", spanFormat))
	}

	sp.logger.Debug("debugSpanProcessor", zap.String("originalFormat", spanFormat), zap.Int("#spans", len(batch.Spans)))
	return 0, nil
}

// NewNoopSpanProcessor creates an OC SpanProcessor that just drops the received data.
func NewNoopSpanProcessor(logger *zap.Logger) SpanProcessor {
	return &debugSpanProcessor{logger: logger}
}
