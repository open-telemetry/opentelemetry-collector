// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	"testing"

	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
)

func TestSpanCount(t *testing.T) {
	md := NewTraceData()
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().Resize(1)
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	assert.EqualValues(t, 1, md.SpanCount())

	rms := md.ResourceSpans()
	rms.Resize(3)
	rms.At(0).InstrumentationLibrarySpans().Resize(1)
	rms.At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	rms.At(1).InstrumentationLibrarySpans().Resize(1)
	rms.At(2).InstrumentationLibrarySpans().Resize(1)
	rms.At(2).InstrumentationLibrarySpans().At(0).Spans().Resize(5)
	assert.EqualValues(t, 6, md.SpanCount())
}

func TestSpanCountWithNils(t *testing.T) {
	assert.EqualValues(t, 0, TraceDataFromOtlp([]*otlptrace.ResourceSpans{nil, {}}).SpanCount())
	assert.EqualValues(t, 0, TraceDataFromOtlp([]*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{nil, {}},
		},
	}).SpanCount())
	assert.EqualValues(t, 2, TraceDataFromOtlp([]*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{nil, {}},
				},
			},
		},
	}).SpanCount())
}

func TestTraceID(t *testing.T) {
	tid := NewTraceID(nil)
	assert.EqualValues(t, []byte(nil), tid.Bytes())

	tid = NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	assert.EqualValues(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, tid.Bytes())
}

func TestToFromOtlp(t *testing.T) {
	otlp := []*otlptrace.ResourceSpans(nil)
	td := TraceDataFromOtlp(otlp)
	assert.EqualValues(t, NewTraceData(), td)
	assert.EqualValues(t, otlp, TraceDataToOtlp(td))
	// More tests in ./tracedata/trace_test.go. Cannot have them here because of
	// circular dependency.
}
