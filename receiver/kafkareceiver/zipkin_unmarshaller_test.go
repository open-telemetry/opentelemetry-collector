// Copyright 2020 The OpenTelemetry Authors
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

package kafkareceiver

import (
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	zipkintranslator "go.opentelemetry.io/collector/translator/trace/zipkin"
)

func TestUnmarshallZipkin(t *testing.T) {
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	td.ResourceSpans().At(0).Resource().Attributes().InitFromMap(
		map[string]pdata.AttributeValue{conventions.AttributeServiceName: pdata.NewAttributeValueString("my_service")})
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetName("foo")
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetStartTime(pdata.TimestampUnixNano(1597759000))
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetEndTime(pdata.TimestampUnixNano(1597769000))
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetTraceID(pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetParentSpanID(pdata.NewSpanID([8]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	spans, err := zipkintranslator.InternalTracesToZipkinSpans(td)
	require.NoError(t, err)

	serializer := zipkinreporter.JSONSerializer{}
	jsonBytes, err := serializer.Serialize(spans)
	require.NoError(t, err)

	tSpan := &zipkincore.Span{Name: "foo"}
	thriftTransport := thrift.NewTMemoryBuffer()
	protocolTransport := thrift.NewTBinaryProtocolTransport(thriftTransport)
	require.NoError(t, protocolTransport.WriteListBegin(thrift.STRUCT, 1))
	err = tSpan.Write(protocolTransport)
	require.NoError(t, err)
	require.NoError(t, protocolTransport.WriteListEnd())

	tdThrift, err := zipkintranslator.V1ThriftBatchToInternalTraces([]*zipkincore.Span{tSpan})
	require.NoError(t, err)

	protoBytes, err := new(zipkin_proto3.SpanSerializer).Serialize(spans)
	require.NoError(t, err)

	tests := []struct {
		unmarshaller Unmarshaller
		encoding     string
		bytes        []byte
		expected     pdata.Traces
	}{
		{
			unmarshaller: zipkinProtoSpanUnmarshaller{},
			encoding:     "zipkin_proto",
			bytes:        protoBytes,
			expected:     td,
		},
		{
			unmarshaller: zipkinJSONSpanUnmarshaller{},
			encoding:     "zipkin_json",
			bytes:        jsonBytes,
			expected:     td,
		},
		{
			unmarshaller: zipkinThriftSpanUnmarshaller{},
			encoding:     "zipkin_thrift",
			bytes:        thriftTransport.Buffer.Bytes(),
			expected:     tdThrift,
		},
	}
	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			traces, err := test.unmarshaller.Unmarshal(test.bytes)
			require.NoError(t, err)
			assert.Equal(t, test.expected, traces)
			assert.Equal(t, test.encoding, test.unmarshaller.Encoding())
		})
	}
}

func TestUnmarshallZipkinThrift_error(t *testing.T) {
	p := zipkinThriftSpanUnmarshaller{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}

func TestUnmarshallZipkinJSON_error(t *testing.T) {
	p := zipkinJSONSpanUnmarshaller{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}

func TestUnmarshallZipkinProto_error(t *testing.T) {
	p := zipkinProtoSpanUnmarshaller{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}
