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
	"testing"

	"github.com/jaegertracing/jaeger/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func TestGetTagFromStatusCode(t *testing.T) {
	tests := []struct {
		name string
		code pdata.StatusCode
		tag  model.KeyValue
	}{
		{
			name: "ok",
			code: pdata.StatusCodeOk,
			tag: model.KeyValue{
				Key:    tracetranslator.TagStatusCode,
				VInt64: int64(pdata.StatusCodeOk),
				VType:  model.ValueType_INT64,
			},
		},

		{
			name: "error",
			code: pdata.StatusCodeError,
			tag: model.KeyValue{
				Key:    tracetranslator.TagStatusCode,
				VInt64: int64(pdata.StatusCodeError),
				VType:  model.ValueType_INT64,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := getTagFromStatusCode(test.code)
			assert.True(t, ok)
			assert.EqualValues(t, test.tag, got)
		})
	}
}

func TestGetErrorTagFromStatusCode(t *testing.T) {
	errTag := model.KeyValue{
		Key:   tracetranslator.TagError,
		VBool: true,
		VType: model.ValueType_BOOL,
	}

	_, ok := getErrorTagFromStatusCode(pdata.StatusCodeUnset)
	assert.False(t, ok)

	_, ok = getErrorTagFromStatusCode(pdata.StatusCodeOk)
	assert.False(t, ok)

	got, ok := getErrorTagFromStatusCode(pdata.StatusCodeError)
	assert.True(t, ok)
	assert.EqualValues(t, errTag, got)
}

func TestGetTagFromStatusMsg(t *testing.T) {
	got, ok := getTagFromStatusMsg("")
	assert.False(t, ok)

	got, ok = getTagFromStatusMsg("test-error")
	assert.True(t, ok)
	assert.EqualValues(t, model.KeyValue{
		Key:   tracetranslator.TagStatusMsg,
		VStr:  "test-error",
		VType: model.ValueType_STRING,
	}, got)
}

func TestGetTagFromSpanKind(t *testing.T) {
	tests := []struct {
		name string
		kind pdata.SpanKind
		tag  model.KeyValue
		ok   bool
	}{
		{
			name: "unspecified",
			kind: pdata.SpanKindUNSPECIFIED,
			tag:  model.KeyValue{},
			ok:   false,
		},

		{
			name: "client",
			kind: pdata.SpanKindCLIENT,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindClient),
			},
			ok: true,
		},

		{
			name: "server",
			kind: pdata.SpanKindSERVER,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindServer),
			},
			ok: true,
		},

		{
			name: "producer",
			kind: pdata.SpanKindPRODUCER,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindProducer),
			},
			ok: true,
		},

		{
			name: "consumer",
			kind: pdata.SpanKindCONSUMER,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindConsumer),
			},
			ok: true,
		},

		{
			name: "internal",
			kind: pdata.SpanKindINTERNAL,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindInternal),
			},
			ok: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := getTagFromSpanKind(test.kind)
			assert.Equal(t, test.ok, ok)
			assert.EqualValues(t, test.tag, got)
		})
	}
}

func TestAttributesToJaegerProtoTags(t *testing.T) {

	attributes := pdata.NewAttributeMap()
	attributes.InsertBool("bool-val", true)
	attributes.InsertInt("int-val", 123)
	attributes.InsertString("string-val", "abc")
	attributes.InsertDouble("double-val", 1.23)
	attributes.InsertString(conventions.AttributeServiceName, "service-name")

	expected := []model.KeyValue{
		{
			Key:   "bool-val",
			VType: model.ValueType_BOOL,
			VBool: true,
		},
		{
			Key:    "int-val",
			VType:  model.ValueType_INT64,
			VInt64: 123,
		},
		{
			Key:   "string-val",
			VType: model.ValueType_STRING,
			VStr:  "abc",
		},
		{
			Key:      "double-val",
			VType:    model.ValueType_FLOAT64,
			VFloat64: 1.23,
		},
		{
			Key:   conventions.AttributeServiceName,
			VType: model.ValueType_STRING,
			VStr:  "service-name",
		},
	}

	got := appendTagsFromAttributes(make([]model.KeyValue, 0, len(expected)), attributes)
	require.EqualValues(t, expected, got)

	// The last item in expected ("service-name") must be skipped in resource tags translation
	got = appendTagsFromResourceAttributes(make([]model.KeyValue, 0, len(expected)-1), attributes)
	require.EqualValues(t, expected[:4], got)
}

func TestInternalTracesToJaegerProto(t *testing.T) {

	tests := []struct {
		name string
		td   pdata.Traces
		jb   *model.Batch
		err  error
	}{
		{
			name: "empty",
			td:   testdata.GenerateTraceDataEmpty(),
			err:  nil,
		},

		{
			name: "no-spans",
			td:   generateTraceDataResourceOnly(),
			jb: &model.Batch{
				Process: generateProtoProcess(),
			},
			err: nil,
		},

		{
			name: "no-resource-attrs",
			td:   generateTraceDataResourceOnlyWithNoAttrs(),
			err:  nil,
		},

		{
			name: "one-span-no-resources",
			td:   generateTraceDataOneSpanNoResourceWithTraceState(),
			jb: &model.Batch{
				Process: &model.Process{
					ServiceName: tracetranslator.ResourceNoServiceName,
				},
				Spans: []*model.Span{
					generateProtoSpanWithTraceState(),
				},
			},
			err: nil,
		},
		{
			name: "library-info",
			td:   generateTraceDataWithLibraryInfo(),
			jb: &model.Batch{
				Process: &model.Process{
					ServiceName: tracetranslator.ResourceNoServiceName,
				},
				Spans: []*model.Span{
					generateProtoSpanWithLibraryInfo("io.opentelemetry.test"),
				},
			},
			err: nil,
		},
		{
			name: "two-spans-child-parent",
			td:   generateTraceDataTwoSpansChildParent(),
			jb: &model.Batch{
				Process: &model.Process{
					ServiceName: tracetranslator.ResourceNoServiceName,
				},
				Spans: []*model.Span{
					generateProtoSpan(),
					generateProtoChildSpanWithErrorTags(),
				},
			},
			err: nil,
		},

		{
			name: "two-spans-with-follower",
			td:   generateTraceDataTwoSpansWithFollower(),
			jb: &model.Batch{
				Process: &model.Process{
					ServiceName: tracetranslator.ResourceNoServiceName,
				},
				Spans: []*model.Span{
					generateProtoSpan(),
					generateProtoFollowerSpan(),
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jbs, err := InternalTracesToJaegerProto(test.td)
			assert.EqualValues(t, test.err, err)
			if test.jb == nil {
				assert.Len(t, jbs, 0)
			} else {
				require.Equal(t, 1, len(jbs))
				assert.EqualValues(t, test.jb, jbs[0])
			}
		})
	}
}

func TestInternalTracesToJaegerProtoBatchesAndBack(t *testing.T) {
	tds, err := goldendataset.GenerateTraces(
		"../../../internal/goldendataset/testdata/generated_pict_pairs_traces.txt",
		"../../../internal/goldendataset/testdata/generated_pict_pairs_spans.txt")
	assert.NoError(t, err)
	for _, td := range tds {
		protoBatches, err := InternalTracesToJaegerProto(td)
		assert.NoError(t, err)
		tdFromPB := ProtoBatchesToInternalTraces(protoBatches)
		assert.NotNil(t, tdFromPB)
		assert.Equal(t, td.SpanCount(), tdFromPB.SpanCount())
	}
}

// generateProtoChildSpanWithErrorTags generates a jaeger span to be used in
// internal->jaeger translation test. It supposed to be the same as generateProtoChildSpan
// that used in jaeger->internal, but jaeger->internal translation infers status code from http status if
// status.code is not set, so the pipeline jaeger->internal->jaeger adds two more tags as the result in that case.
func generateProtoChildSpanWithErrorTags() *model.Span {
	span := generateProtoChildSpan()
	span.Tags = append(span.Tags, model.KeyValue{
		Key:    tracetranslator.TagStatusCode,
		VType:  model.ValueType_INT64,
		VInt64: int64(pdata.StatusCodeError),
	})
	span.Tags = append(span.Tags, model.KeyValue{
		Key:   tracetranslator.TagError,
		VBool: true,
		VType: model.ValueType_BOOL,
	})
	return span
}

func BenchmarkInternalTracesToJaegerProto(b *testing.B) {
	td := generateTraceDataTwoSpansChildParent()
	resource := generateTraceDataResourceOnly().ResourceSpans().At(0).Resource()
	resource.CopyTo(td.ResourceSpans().At(0).Resource())

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		InternalTracesToJaegerProto(td)
	}
}
