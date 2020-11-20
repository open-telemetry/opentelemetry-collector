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

package jaeger

import (
	"encoding/binary"
	"testing"

	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func TestJThriftTagsToInternalAttributes(t *testing.T) {
	var intVal int64 = 123
	boolVal := true
	stringVal := "abc"
	doubleVal := 1.23
	tags := []*jaeger.Tag{
		{
			Key:   "bool-val",
			VType: jaeger.TagType_BOOL,
			VBool: &boolVal,
		},
		{
			Key:   "int-val",
			VType: jaeger.TagType_LONG,
			VLong: &intVal,
		},
		{
			Key:   "string-val",
			VType: jaeger.TagType_STRING,
			VStr:  &stringVal,
		},
		{
			Key:     "double-val",
			VType:   jaeger.TagType_DOUBLE,
			VDouble: &doubleVal,
		},
		{
			Key:     "binary-val",
			VType:   jaeger.TagType_BINARY,
			VBinary: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
		},
	}

	expected := pdata.NewAttributeMap()
	expected.InsertBool("bool-val", true)
	expected.InsertInt("int-val", 123)
	expected.InsertString("string-val", "abc")
	expected.InsertDouble("double-val", 1.23)
	expected.InsertString("binary-val", "AAAAAABkfZg=")

	got := pdata.NewAttributeMap()
	jThriftTagsToInternalAttributes(tags, got)

	require.EqualValues(t, expected, got)
}

func TestThriftBatchToInternalTraces(t *testing.T) {

	tests := []struct {
		name string
		jb   *jaeger.Batch
		td   pdata.Traces
	}{
		{
			name: "empty",
			jb:   &jaeger.Batch{},
			td:   testdata.GenerateTraceDataEmpty(),
		},

		{
			name: "no-spans",
			jb: &jaeger.Batch{
				Process: generateThriftProcess(),
			},
			td: testdata.GenerateTraceDataNoLibraries(),
		},

		{
			name: "one-span-no-resources",
			jb: &jaeger.Batch{
				Spans: []*jaeger.Span{
					generateThriftSpan(),
				},
			},
			td: generateTraceDataOneSpanNoResource(),
		},
		{
			name: "two-spans-child-parent",
			jb: &jaeger.Batch{
				Spans: []*jaeger.Span{
					generateThriftSpan(),
					generateThriftChildSpan(),
				},
			},
			td: generateTraceDataTwoSpansChildParent(),
		},

		{
			name: "two-spans-with-follower",
			jb: &jaeger.Batch{
				Spans: []*jaeger.Span{
					generateThriftSpan(),
					generateThriftFollowerSpan(),
				},
			},
			td: generateTraceDataTwoSpansWithFollower(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			td := ThriftBatchToInternalTraces(test.jb)
			assert.EqualValues(t, test.td, td)
		})
	}
}

func generateThriftProcess() *jaeger.Process {
	attrVal := "resource-attr-val-1"
	return &jaeger.Process{
		Tags: []*jaeger.Tag{
			{
				Key:   "resource-attr",
				VType: jaeger.TagType_STRING,
				VStr:  &attrVal,
			},
		},
	}
}

func generateThriftSpan() *jaeger.Span {
	spanStartTs := unixNanoToMicroseconds(testSpanStartTimestamp)
	spanEndTs := unixNanoToMicroseconds(testSpanEndTimestamp)
	eventTs := unixNanoToMicroseconds(testSpanEventTimestamp)
	intAttrVal := int64(123)
	eventName := "event-with-attr"
	eventStrAttrVal := "span-event-attr-val"
	statusCode := int64(pdata.StatusCodeError)
	statusMsg := "status-cancelled"
	kind := string(tracetranslator.OpenTracingSpanKindClient)

	return &jaeger.Span{
		TraceIdHigh:   int64(binary.BigEndian.Uint64([]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8})),
		TraceIdLow:    int64(binary.BigEndian.Uint64([]byte{0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80})),
		SpanId:        int64(binary.BigEndian.Uint64([]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8})),
		OperationName: "operationA",
		StartTime:     spanStartTs,
		Duration:      spanEndTs - spanStartTs,
		Logs: []*jaeger.Log{
			{
				Timestamp: eventTs,
				Fields: []*jaeger.Tag{
					{
						Key:   tracetranslator.TagMessage,
						VType: jaeger.TagType_STRING,
						VStr:  &eventName,
					},
					{
						Key:   "span-event-attr",
						VType: jaeger.TagType_STRING,
						VStr:  &eventStrAttrVal,
					},
				},
			},
			{
				Timestamp: eventTs,
				Fields: []*jaeger.Tag{
					{
						Key:   "attr-int",
						VType: jaeger.TagType_LONG,
						VLong: &intAttrVal,
					},
				},
			},
		},
		Tags: []*jaeger.Tag{
			{
				Key:   tracetranslator.TagStatusCode,
				VType: jaeger.TagType_LONG,
				VLong: &statusCode,
			},
			{
				Key:   tracetranslator.TagStatusMsg,
				VType: jaeger.TagType_STRING,
				VStr:  &statusMsg,
			},
			{
				Key:   tracetranslator.TagSpanKind,
				VType: jaeger.TagType_STRING,
				VStr:  &kind,
			},
		},
	}
}

func generateThriftChildSpan() *jaeger.Span {
	spanStartTs := unixNanoToMicroseconds(testSpanStartTimestamp)
	spanEndTs := unixNanoToMicroseconds(testSpanEndTimestamp)
	notFoundAttrVal := int64(404)
	kind := string(tracetranslator.OpenTracingSpanKindServer)

	return &jaeger.Span{
		TraceIdHigh:   int64(binary.BigEndian.Uint64([]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8})),
		TraceIdLow:    int64(binary.BigEndian.Uint64([]byte{0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80})),
		SpanId:        int64(binary.BigEndian.Uint64([]byte{0x1F, 0x1E, 0x1D, 0x1C, 0x1B, 0x1A, 0x19, 0x18})),
		ParentSpanId:  int64(binary.BigEndian.Uint64([]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8})),
		OperationName: "operationB",
		StartTime:     spanStartTs,
		Duration:      spanEndTs - spanStartTs,
		Tags: []*jaeger.Tag{
			{
				Key:   tracetranslator.TagHTTPStatusCode,
				VType: jaeger.TagType_LONG,
				VLong: &notFoundAttrVal,
			},
			{
				Key:   tracetranslator.TagSpanKind,
				VType: jaeger.TagType_STRING,
				VStr:  &kind,
			},
		},
	}
}

func generateThriftFollowerSpan() *jaeger.Span {
	statusCode := int64(pdata.StatusCodeOk)
	statusMsg := "status-ok"
	kind := string(tracetranslator.OpenTracingSpanKindConsumer)

	return &jaeger.Span{
		TraceIdHigh:   int64(binary.BigEndian.Uint64([]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8})),
		TraceIdLow:    int64(binary.BigEndian.Uint64([]byte{0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80})),
		SpanId:        int64(binary.BigEndian.Uint64([]byte{0x1F, 0x1E, 0x1D, 0x1C, 0x1B, 0x1A, 0x19, 0x18})),
		OperationName: "operationC",
		StartTime:     unixNanoToMicroseconds(testSpanEndTimestamp),
		Duration:      1000,
		Tags: []*jaeger.Tag{
			{
				Key:   tracetranslator.TagStatusCode,
				VType: jaeger.TagType_LONG,
				VLong: &statusCode,
			},
			{
				Key:   tracetranslator.TagStatusMsg,
				VType: jaeger.TagType_STRING,
				VStr:  &statusMsg,
			},
			{
				Key:   tracetranslator.TagSpanKind,
				VType: jaeger.TagType_STRING,
				VStr:  &kind,
			},
		},
		References: []*jaeger.SpanRef{
			{
				TraceIdHigh: int64(binary.BigEndian.Uint64([]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8})),
				TraceIdLow:  int64(binary.BigEndian.Uint64([]byte{0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80})),
				SpanId:      int64(binary.BigEndian.Uint64([]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8})),
				RefType:     jaeger.SpanRefType_FOLLOWS_FROM,
			},
		},
	}
}

func unixNanoToMicroseconds(ns pdata.TimestampUnixNano) int64 {
	return int64(ns / 1000)
}

func BenchmarkThriftBatchToInternalTraces(b *testing.B) {
	jb := &jaeger.Batch{
		Process: generateThriftProcess(),
		Spans: []*jaeger.Span{
			generateThriftSpan(),
			generateThriftChildSpan(),
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ThriftBatchToInternalTraces(jb)
	}
}
