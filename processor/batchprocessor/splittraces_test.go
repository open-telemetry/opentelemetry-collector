// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batchprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestSplitTraces_noop(t *testing.T) {
	td := testdata.GenerateTraceDataManySpansSameResource(20)
	splitSize := 40
	split := splitTrace(splitSize, td)
	assert.Equal(t, td, split)

	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(5)
	assert.EqualValues(t, td, split)
}

func TestSplitTraces(t *testing.T) {
	td := testdata.GenerateTraceDataManySpansSameResource(20)
	spans := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
	for i := 0; i < spans.Len(); i++ {
		spans.At(i).SetName(getTestSpanName(0, i))
	}
	cp := pdata.NewTraces()
	cp.ResourceSpans().Resize(1)
	cp.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	cp.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(5)
	cpSpans := cp.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
	td.ResourceSpans().At(0).Resource().CopyTo(
		cp.ResourceSpans().At(0).Resource())
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).InstrumentationLibrary().CopyTo(
		cp.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).InstrumentationLibrary())
	spans.At(19).CopyTo(cpSpans.At(0))
	spans.At(18).CopyTo(cpSpans.At(1))
	spans.At(17).CopyTo(cpSpans.At(2))
	spans.At(16).CopyTo(cpSpans.At(3))
	spans.At(15).CopyTo(cpSpans.At(4))

	splitSize := 5
	split := splitTrace(splitSize, td)
	assert.Equal(t, splitSize, split.SpanCount())
	assert.Equal(t, cp, split)
	assert.Equal(t, 15, td.SpanCount())
	assert.Equal(t, "test-span-0-19", split.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
	assert.Equal(t, "test-span-0-15", split.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(4).Name())
}

func TestSplitTracesMultipleResourceSpans(t *testing.T) {
	td := testdata.GenerateTraceDataManySpansSameResource(20)
	spans := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
	for i := 0; i < spans.Len(); i++ {
		spans.At(i).SetName(getTestSpanName(0, i))
	}
	td.ResourceSpans().Resize(2)
	// add second index to resource spans
	testdata.GenerateTraceDataManySpansSameResource(20).
		ResourceSpans().At(0).CopyTo(td.ResourceSpans().At(1))
	spans = td.ResourceSpans().At(1).InstrumentationLibrarySpans().At(0).Spans()
	for i := 0; i < spans.Len(); i++ {
		spans.At(i).SetName(getTestSpanName(1, i))
	}

	splitSize := 5
	split := splitTrace(splitSize, td)
	assert.Equal(t, splitSize, split.SpanCount())
	assert.Equal(t, 35, td.SpanCount())
	assert.Equal(t, "test-span-1-19", split.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
	assert.Equal(t, "test-span-1-15", split.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(4).Name())
}

func TestSplitTracesMultipleResourceSpans_split_size_greater_than_span_size(t *testing.T) {
	td := testdata.GenerateTraceDataManySpansSameResource(20)
	spans := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
	for i := 0; i < spans.Len(); i++ {
		spans.At(i).SetName(getTestSpanName(0, i))
	}
	td.ResourceSpans().Resize(2)
	// add second index to resource spans
	testdata.GenerateTraceDataManySpansSameResource(20).
		ResourceSpans().At(0).CopyTo(td.ResourceSpans().At(1))
	spans = td.ResourceSpans().At(1).InstrumentationLibrarySpans().At(0).Spans()
	for i := 0; i < spans.Len(); i++ {
		spans.At(i).SetName(getTestSpanName(1, i))
	}

	splitSize := 25
	split := splitTrace(splitSize, td)
	assert.Equal(t, splitSize, split.SpanCount())
	assert.Equal(t, 40-splitSize, td.SpanCount())
	assert.Equal(t, 1, td.ResourceSpans().Len())
	assert.Equal(t, "test-span-1-19", split.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
	assert.Equal(t, "test-span-1-0", split.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(19).Name())
	assert.Equal(t, "test-span-0-19", split.ResourceSpans().At(1).InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
	assert.Equal(t, "test-span-0-15", split.ResourceSpans().At(1).InstrumentationLibrarySpans().At(0).Spans().At(4).Name())
}
