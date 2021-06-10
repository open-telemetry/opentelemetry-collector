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
	"context"
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	zipkintranslator "go.opentelemetry.io/collector/translator/trace/zipkin"
	"go.opentelemetry.io/collector/translator/trace/zipkinv2"
)

var fromTranslator zipkinv2.FromTranslator

func TestUnmarshalZipkin(t *testing.T) {
	td := pdata.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().InitFromMap(
		map[string]pdata.AttributeValue{conventions.AttributeServiceName: pdata.NewAttributeValueString("my_service")})
	span := rs.InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("foo")
	span.SetStartTimestamp(pdata.Timestamp(1597759000))
	span.SetEndTimestamp(pdata.Timestamp(1597769000))
	span.SetTraceID(pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	span.SetParentSpanID(pdata.NewSpanID([8]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	ret, err := fromTranslator.FromTraces(td)
	require.NoError(t, err)
	spans := ret.([]*zipkinmodel.SpanModel)

	serializer := zipkinreporter.JSONSerializer{}
	jsonBytes, err := serializer.Serialize(spans)
	require.NoError(t, err)

	tSpan := &zipkincore.Span{Name: "foo"}
	thriftTransport := thrift.NewTMemoryBuffer()
	protocolTransport := thrift.NewTBinaryProtocolConf(thriftTransport, nil)
	require.NoError(t, protocolTransport.WriteListBegin(context.Background(), thrift.STRUCT, 1))
	err = tSpan.Write(context.Background(), protocolTransport)
	require.NoError(t, err)
	require.NoError(t, protocolTransport.WriteListEnd(context.Background()))

	tdThrift, err := zipkintranslator.V1ThriftBatchToInternalTraces([]*zipkincore.Span{tSpan})
	require.NoError(t, err)

	protoBytes, err := new(zipkin_proto3.SpanSerializer).Serialize(spans)
	require.NoError(t, err)

	tests := []struct {
		unmarshaler TracesUnmarshaler
		encoding    string
		bytes       []byte
		expected    pdata.Traces
	}{
		{
			unmarshaler: zipkinProtoSpanUnmarshaler{},
			encoding:    "zipkin_proto",
			bytes:       protoBytes,
			expected:    td,
		},
		{
			unmarshaler: zipkinJSONSpanUnmarshaler{},
			encoding:    "zipkin_json",
			bytes:       jsonBytes,
			expected:    td,
		},
		{
			unmarshaler: zipkinThriftSpanUnmarshaler{},
			encoding:    "zipkin_thrift",
			bytes:       thriftTransport.Buffer.Bytes(),
			expected:    tdThrift,
		},
	}
	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			traces, err := test.unmarshaler.Unmarshal(test.bytes)
			require.NoError(t, err)
			assert.Equal(t, test.expected, traces)
			assert.Equal(t, test.encoding, test.unmarshaler.Encoding())
		})
	}
}

func TestUnmarshalZipkinThrift_error(t *testing.T) {
	p := zipkinThriftSpanUnmarshaler{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}

func TestUnmarshalZipkinJSON_error(t *testing.T) {
	p := zipkinJSONSpanUnmarshaler{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}

func TestUnmarshalZipkinProto_error(t *testing.T) {
	p := zipkinProtoSpanUnmarshaler{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}
