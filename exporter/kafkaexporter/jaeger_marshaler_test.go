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

package kafkaexporter

import (
	"bytes"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

func TestJaegerMarshaler(t *testing.T) {
	td := pdata.NewTraces()
	span := td.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("foo")
	span.SetStartTimestamp(pdata.Timestamp(10))
	span.SetEndTimestamp(pdata.Timestamp(20))
	span.SetTraceID(pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	batches, err := jaegertranslator.InternalTracesToJaegerProto(td)
	require.NoError(t, err)

	batches[0].Spans[0].Process = batches[0].Process
	jaegerProtoBytes, err := batches[0].Spans[0].Marshal()
	messageKey := []byte(batches[0].Spans[0].TraceID.String())
	require.NoError(t, err)
	require.NotNil(t, jaegerProtoBytes)

	jsonMarshaler := &jsonpb.Marshaler{}
	jsonByteBuffer := new(bytes.Buffer)
	require.NoError(t, jsonMarshaler.Marshal(jsonByteBuffer, batches[0].Spans[0]))

	tests := []struct {
		unmarshaler TracesMarshaler
		encoding    string
		messages    []*sarama.ProducerMessage
	}{
		{
			unmarshaler: jaegerMarshaler{
				marshaler: jaegerProtoSpanMarshaler{},
			},
			encoding: "jaeger_proto",
			messages: []*sarama.ProducerMessage{{Topic: "topic", Value: sarama.ByteEncoder(jaegerProtoBytes), Key: sarama.ByteEncoder(messageKey)}},
		},
		{
			unmarshaler: jaegerMarshaler{
				marshaler: jaegerJSONSpanMarshaler{
					pbMarshaler: &jsonpb.Marshaler{},
				},
			},
			encoding: "jaeger_json",
			messages: []*sarama.ProducerMessage{{Topic: "topic", Value: sarama.ByteEncoder(jsonByteBuffer.Bytes()), Key: sarama.ByteEncoder(messageKey)}},
		},
	}
	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			messages, err := test.unmarshaler.Marshal(td, "topic")
			require.NoError(t, err)
			assert.Equal(t, test.messages, messages)
			assert.Equal(t, test.encoding, test.unmarshaler.Encoding())
		})
	}
}

func TestJaegerMarshaler_error_covert_traceID(t *testing.T) {
	marshaler := jaegerMarshaler{
		marshaler: jaegerProtoSpanMarshaler{},
	}
	td := pdata.NewTraces()
	td.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	// fails in zero traceID
	messages, err := marshaler.Marshal(td, "topic")
	require.Error(t, err)
	assert.Nil(t, messages)
}
