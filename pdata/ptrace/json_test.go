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

package ptrace

import (
	"fmt"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

var tracesOTLP = func() Traces {
	td := NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	il := rs.ScopeSpans().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Spans().AppendEmpty().SetName("testSpan")
	return td
}()

var tracesJSON = `{"resourceSpans":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"scopeSpans":[{"scope":{"name":"name","version":"version"},"spans":[{"traceId":"","spanId":"","parentSpanId":"","name":"testSpan","status":{}}]}]}]}`

func TestTracesJSON(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)

	decoder := NewJSONUnmarshaler()
	var got interface{}
	got, err = decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, tracesOTLP, got)
}

func TestTracesJSON_Marshal(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)
	assert.Equal(t, tracesJSON, string(jsonBuf))
}

var tracesOTLPFull = func() Traces {
	traceID := internal.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := internal.NewSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	td := NewTraces()
	// Add ResourceSpans.
	rs := td.ResourceSpans().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	rs.Resource().Attributes().UpsertString("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add InstrumentationLibrarySpans.
	il := rs.ScopeSpans().AppendEmpty()
	il.Scope().SetName("instrumentation name")
	il.Scope().SetVersion("instrumentation version")
	il.SetSchemaUrl("schemaURL")
	// Add spans.
	sp := il.Spans().AppendEmpty()
	sp.SetName("testSpan")
	sp.SetKind(internal.SpanKindClient)
	sp.SetDroppedAttributesCount(1)
	sp.SetStartTimestamp(internal.NewTimestampFromTime(time.Now()))
	sp.SetTraceID(traceID)
	sp.SetSpanID(spanID)
	sp.SetDroppedEventsCount(1)
	sp.SetDroppedLinksCount(1)
	sp.SetEndTimestamp(internal.NewTimestampFromTime(time.Now()))
	sp.SetParentSpanID(spanID)
	sp.SetTraceState("state")
	sp.Status().SetCode(internal.StatusCodeOk)
	sp.Status().SetMessage("message")
	// Add attributes.
	sp.Attributes().UpsertString("string", "value")
	sp.Attributes().UpsertBool("bool", true)
	sp.Attributes().UpsertInt("int", 1)
	sp.Attributes().UpsertDouble("double", 1.1)
	sp.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	arr := internal.NewValueSlice()
	arr.SliceVal().AppendEmpty().SetIntVal(1)
	arr.SliceVal().AppendEmpty().SetStringVal("str")
	sp.Attributes().Upsert("array", arr)
	kvList := internal.NewValueMap()
	kvList.MapVal().Upsert("int", internal.NewValueInt(1))
	kvList.MapVal().Upsert("string", internal.NewValueString("string"))
	sp.Attributes().Upsert("kvList", kvList)
	// Add events.
	event := sp.Events().AppendEmpty()
	event.SetName("eventName")
	event.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	event.SetDroppedAttributesCount(1)
	event.Attributes().UpsertString("string", "value")
	event.Attributes().UpsertBool("bool", true)
	event.Attributes().UpsertInt("int", 1)
	event.Attributes().UpsertDouble("double", 1.1)
	event.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	// Add links.
	link := sp.Links().AppendEmpty()
	link.SetTraceState("state")
	link.SetTraceID(traceID)
	link.SetSpanID(spanID)
	link.SetDroppedAttributesCount(1)
	link.Attributes().UpsertString("string", "value")
	link.Attributes().UpsertBool("bool", true)
	link.Attributes().UpsertInt("int", 1)
	link.Attributes().UpsertDouble("double", 1.1)
	link.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	// Add another span.
	sp2 := il.Spans().AppendEmpty()
	sp2.SetName("testSpan2")
	return td
}()

func TestJSONFull(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLPFull)
	assert.NoError(t, err)

	decoder := NewJSONUnmarshaler()
	got, err := decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)
	assert.EqualValues(t, tracesOTLPFull, got)
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()

	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLPFull)
	assert.NoError(b, err)
	decoder := NewJSONUnmarshaler()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := decoder.UnmarshalTraces(jsonBuf)
			assert.NoError(b, err)
		}
	})
}

func TestReadInt64(t *testing.T) {
	var data = `{"intAsNumber":1,"intAsString":"1"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(data))
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "intAsNumber":
			v := readInt64(iter)
			assert.Equal(t, int64(1), v)
		case "intAsString":
			v := readInt64(iter)
			assert.Equal(t, int64(1), v)
		}
		return true
	})
	assert.NoError(t, iter.Error)
}

func TestReadTraceDataUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readTraceData(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadResourceSpansUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readResourceSpans(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadResourceSpansUnknownResourceField(t *testing.T) {
	jsonStr := `{"resource":{"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readResourceSpans(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadInstrumentationLibrarySpansUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readInstrumentationLibrarySpans(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadInstrumentationLibrarySpansUnknownInstrumentationLibraryField(t *testing.T) {
	jsonStr := `{"instrumentationLibrary":{"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readInstrumentationLibrarySpans(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadSpanUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadSpanUnknownStatusField(t *testing.T) {
	jsonStr := `{"status":{"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}
func TestReadSpanInvalidTraceIDField(t *testing.T) {
	jsonStr := `{"trace_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}
func TestReadSpanInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}
func TestReadSpanInvalidParentSpanIDField(t *testing.T) {
	jsonStr := `{"parent_span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse parent_span_id")
	}
}

func TestReadSpanLinkUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpanLink(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadSpanLinkInvalidTraceIDField(t *testing.T) {
	jsonStr := `{"trace_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpanLink(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestReadSpanLinkInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpanLink(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestReadSpanEventUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpanEvent(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadAttributeUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAttribute(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadAnyValueUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAnyValue(iter, "")
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadAnyValueInvliadBytesValue(t *testing.T) {
	jsonStr := `"--"`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAnyValue(iter, "bytesValue")
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "base64")
	}
}

func TestReadArrayUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readArray(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadKvlistValueUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readKvlistValue(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadSpanKind(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    otlptrace.Span_SpanKind
	}{
		{
			name:    "string",
			jsonStr: fmt.Sprintf(`"%s"`, otlptrace.Span_SPAN_KIND_INTERNAL.String()),
			want:    otlptrace.Span_SPAN_KIND_INTERNAL,
		},
		{
			name:    "int",
			jsonStr: fmt.Sprintf("%d", otlptrace.Span_SPAN_KIND_INTERNAL),
			want:    otlptrace.Span_SPAN_KIND_INTERNAL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			if got := readSpanKind(iter); got != tt.want {
				t.Errorf("readSpanKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadStatusCode(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    otlptrace.Status_StatusCode
	}{
		{
			name:    "string",
			jsonStr: fmt.Sprintf(`"%s"`, otlptrace.Status_STATUS_CODE_ERROR.String()),
			want:    otlptrace.Status_STATUS_CODE_ERROR,
		},
		{
			name:    "int",
			jsonStr: fmt.Sprintf("%d", otlptrace.Status_STATUS_CODE_ERROR),
			want:    otlptrace.Status_STATUS_CODE_ERROR,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			if got := readStatusCode(iter); got != tt.want {
				t.Errorf("readStatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}
