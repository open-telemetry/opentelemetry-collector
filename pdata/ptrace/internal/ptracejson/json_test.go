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

package ptracejson

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

func TestReadTraceDataUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	value := &otlptrace.TracesData{}
	assert.NoError(t, UnmarshalTraceData([]byte(jsonStr), value))
	assert.Equal(t, &otlptrace.TracesData{}, value)
}

func TestReadExportTraceServiceRequestUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	value := &otlpcollectortrace.ExportTraceServiceRequest{}
	assert.NoError(t, UnmarshalExportTraceServiceRequest([]byte(jsonStr), value))
	assert.Equal(t, &otlpcollectortrace.ExportTraceServiceRequest{}, value)
}

func TestReadResourceSpansUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readResourceSpans(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.ResourceSpans{}, val)
}

func TestReadScopeSpansUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readScopeSpans(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.ScopeSpans{}, val)
}

func TestReadSpanUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readSpan(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span{}, val)
}

func TestReadSpanUnknownStatusField(t *testing.T) {
	jsonStr := `{"status":{"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readSpan(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span{}, val)
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
	val := readSpanLink(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span_Link{}, val)
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
	val := readSpanEvent(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span_Event{}, val)
}
