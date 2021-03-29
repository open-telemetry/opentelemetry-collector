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
	"bytes"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

func TestUnmarshallJaeger(t *testing.T) {
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

	protoBytes, err := batches[0].Spans[0].Marshal()
	require.NoError(t, err)

	jsonMarshaller := &jsonpb.Marshaler{}
	jsonBytes := new(bytes.Buffer)
	jsonMarshaller.Marshal(jsonBytes, batches[0].Spans[0])

	tests := []struct {
		unmarshaller Unmarshaller
		encoding     string
		bytes        []byte
	}{
		{
			unmarshaller: jaegerProtoSpanUnmarshaller{},
			encoding:     "jaeger_proto",
			bytes:        protoBytes,
		},
		{
			unmarshaller: jaegerJSONSpanUnmarshaller{},
			encoding:     "jaeger_json",
			bytes:        jsonBytes.Bytes(),
		},
	}
	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			got, err := test.unmarshaller.Unmarshal(test.bytes)
			require.NoError(t, err)
			assert.Equal(t, td, got)
			assert.Equal(t, test.encoding, test.unmarshaller.Encoding())
		})
	}
}

func TestUnmarshallJaegerProto_error(t *testing.T) {
	p := jaegerProtoSpanUnmarshaller{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}

func TestUnmarshallJaegerJSON_error(t *testing.T) {
	p := jaegerJSONSpanUnmarshaller{}
	got, err := p.Unmarshal([]byte("+$%"))
	assert.Equal(t, pdata.NewTraces(), got)
	assert.Error(t, err)
}
