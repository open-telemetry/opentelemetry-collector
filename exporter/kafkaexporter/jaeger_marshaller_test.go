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

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

func TestJaegerMarshaller(t *testing.T) {
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetName("foo")
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetStartTime(pdata.Timestamp(10))
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetEndTime(pdata.Timestamp(20))
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetTraceID(pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	batches, err := jaegertranslator.InternalTracesToJaegerProto(td)
	require.NoError(t, err)

	batches[0].Spans[0].Process = batches[0].Process
	jaegerProtoBytes, err := batches[0].Spans[0].Marshal()
	require.NoError(t, err)
	require.NotNil(t, jaegerProtoBytes)

	jsonMarshaller := &jsonpb.Marshaler{}
	jsonByteBuffer := new(bytes.Buffer)
	require.NoError(t, jsonMarshaller.Marshal(jsonByteBuffer, batches[0].Spans[0]))

	tests := []struct {
		unmarshaller TracesMarshaller
		encoding     string
		messages     []Message
	}{
		{
			unmarshaller: jaegerMarshaller{
				marshaller: jaegerProtoSpanMarshaller{},
			},
			encoding: "jaeger_proto",
			messages: []Message{{Value: jaegerProtoBytes}},
		},
		{
			unmarshaller: jaegerMarshaller{
				marshaller: jaegerJSONSpanMarshaller{
					pbMarshaller: &jsonpb.Marshaler{},
				},
			},
			encoding: "jaeger_json",
			messages: []Message{{Value: jsonByteBuffer.Bytes()}},
		},
	}
	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			messages, err := test.unmarshaller.Marshal(td)
			require.NoError(t, err)
			assert.Equal(t, test.messages, messages)
			assert.Equal(t, test.encoding, test.unmarshaller.Encoding())
		})
	}
}

func TestJaegerMarshaller_error_covert_traceID(t *testing.T) {
	marshaller := jaegerMarshaller{
		marshaller: jaegerProtoSpanMarshaller{},
	}
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	// fails in zero traceID
	messages, err := marshaller.Marshal(td)
	require.Error(t, err)
	assert.Nil(t, messages)
}
