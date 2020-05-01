// Copyright 2020, OpenTelemetry Authors
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

package jaeger

import (
	"testing"

	"github.com/jaegertracing/jaeger/model"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestAppendTagFromSpanStatus(t *testing.T) {
	okStatus := pdata.NewSpanStatus()
	okStatus.InitEmpty()
	okStatus.SetCode(pdata.StatusCode(otlptrace.Status_Ok))

	unknownStatus := pdata.NewSpanStatus()
	unknownStatus.InitEmpty()
	unknownStatus.SetCode(pdata.StatusCode(otlptrace.Status_UnknownError))

	notFoundStatus := pdata.NewSpanStatus()
	notFoundStatus.InitEmpty()
	notFoundStatus.SetCode(pdata.StatusCode(otlptrace.Status_NotFound))
	notFoundStatus.SetMessage("404 Not Found")

	tests := []struct {
		name   string
		status pdata.SpanStatus
		tags   []model.KeyValue
	}{
		{
			name:   "none",
			status: pdata.NewSpanStatus(),
			tags:   []model.KeyValue(nil),
		},

		{
			name:   "ok",
			status: okStatus,
			tags: []model.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(otlptrace.Status_Ok),
					VType:  model.ValueType_INT64,
				},
			},
		},

		{
			name:   "unknown",
			status: unknownStatus,
			tags: []model.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(otlptrace.Status_UnknownError),
					VType:  model.ValueType_INT64,
				},
				{
					Key:   tracetranslator.TagError,
					VBool: true,
					VType: model.ValueType_BOOL,
				},
			},
		},

		{
			name:   "not-found-with-msg",
			status: notFoundStatus,
			tags: []model.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(otlptrace.Status_NotFound),
					VType:  model.ValueType_INT64,
				},
				{
					Key:   tracetranslator.TagError,
					VBool: true,
					VType: model.ValueType_BOOL,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  "404 Not Found",
					VType: model.ValueType_STRING,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tags := appendTagFromSpanStatus(nil, test.status)
			assert.EqualValues(t, test.tags, tags)
		})
	}
}

func TestAppendTagFromSpanKind(t *testing.T) {
	tests := []struct {
		name string
		kind pdata.SpanKind
		tags []model.KeyValue
	}{
		{
			name: "unspecified",
			kind: pdata.SpanKindUNSPECIFIED,
			tags: []model.KeyValue(nil),
		},

		{
			name: "client",
			kind: pdata.SpanKindCLIENT,
			tags: []model.KeyValue{
				{
					Key:   tracetranslator.TagSpanKind,
					VType: model.ValueType_STRING,
					VStr:  string(tracetranslator.OpenTracingSpanKindClient),
				},
			},
		},

		{
			name: "server",
			kind: pdata.SpanKindSERVER,
			tags: []model.KeyValue{
				{
					Key:   tracetranslator.TagSpanKind,
					VType: model.ValueType_STRING,
					VStr:  string(tracetranslator.OpenTracingSpanKindServer),
				},
			},
		},

		{
			name: "producer",
			kind: pdata.SpanKindPRODUCER,
			tags: []model.KeyValue{
				{
					Key:   tracetranslator.TagSpanKind,
					VType: model.ValueType_STRING,
					VStr:  string(tracetranslator.OpenTracingSpanKindProducer),
				},
			},
		},

		{
			name: "consumer",
			kind: pdata.SpanKindCONSUMER,
			tags: []model.KeyValue{
				{
					Key:   tracetranslator.TagSpanKind,
					VType: model.ValueType_STRING,
					VStr:  string(tracetranslator.OpenTracingSpanKindConsumer),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tags := appendTagFromSpanKind(nil, test.kind)
			assert.EqualValues(t, test.tags, tags)
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

	require.EqualValues(t, expected, attributesToJaegerProtoTags(attributes))

	got := attributesToJaegerProtoTags(attributes)
	require.EqualValues(t, expected, got)

	// The last item in expected ("service-name") must be skipped in resource tags translation
	got = resourceAttributesToJaegerProtoTags(attributes)
	require.EqualValues(t, expected[:4], got)
}

func TestInternalTracesToJaegerProto(t *testing.T) {

	tests := []struct {
		name string
		td   pdata.Traces
		jb   model.Batch
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
			jb: model.Batch{
				Process: generateProtoProcess(),
			},
			err: nil,
		},

		{
			name: "one-span-no-resources",
			td:   generateTraceDataOneSpanNoResource(),
			jb: model.Batch{
				Spans: []*model.Span{
					generateProtoSpan(),
				},
			},
			err: nil,
		},
		{
			name: "two-spans-child-parent",
			td:   generateTraceDataTwoSpansChildParent(),
			jb: model.Batch{
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
			jb: model.Batch{
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
			if test.name == "empty" {
				assert.Nil(t, jbs)
			} else {
				assert.Equal(t, 1, len(jbs))
				assert.EqualValues(t, test.jb, *jbs[0])
			}
		})
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
		VInt64: tracetranslator.OCNotFound,
	})
	span.Tags = append(span.Tags, model.KeyValue{
		Key:   tracetranslator.TagError,
		VBool: true,
		VType: model.ValueType_BOOL,
	})
	return span
}
