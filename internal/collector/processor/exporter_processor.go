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

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/processor"
)

type exporterSpanProcessor struct {
	tdp processor.TraceDataProcessor
}

var _ SpanProcessor = (*exporterSpanProcessor)(nil)

// NewTraceExporterProcessor creates processor that feeds SpanData to the given trace exporters.
func NewTraceExporterProcessor(traceExporters ...processor.TraceDataProcessor) SpanProcessor {
	return &exporterSpanProcessor{tdp: processor.NewMultiTraceDataProcessor(traceExporters)}
}

func (sp *exporterSpanProcessor) ProcessSpans(td data.TraceData, spanFormat string) error {
	err := sp.tdp.ProcessTraceData(context.Background(), td)
	if err != nil {
		return err
	}
	return nil
}
