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

package zipkinv2

import (
	"io/ioutil"
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestProtobufDecoder_DecodeTraces(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/zipkin_v2_single.json")
	require.NoError(t, err)
	decoder := jsonDecoder{}
	spans, err := decoder.DecodeTraces(data)
	require.NoError(t, err)

	pb, err := zipkin_proto3.SpanSerializer{}.Serialize(spans.([]*zipkinmodel.SpanModel))
	require.NoError(t, err)

	pbDecoder := protobufDecoder{DebugWasSet: false}
	pbSpans, err := pbDecoder.DecodeTraces(pb)
	assert.NoError(t, err)
	assert.IsType(t, []*zipkinmodel.SpanModel{}, pbSpans)
	assert.Len(t, pbSpans, 1)
}

func TestProtobufDecoder_DecodeTracesError(t *testing.T) {
	decoder := protobufDecoder{DebugWasSet: false}
	spans, err := decoder.DecodeTraces([]byte("{"))
	assert.Error(t, err)
	assert.Nil(t, spans)
}

func TestProtobufEncoder_EncodeTraces(t *testing.T) {
	encoder := protobufEncoder{}
	buf, err := encoder.EncodeTraces(generateSpanErrorTags())
	assert.NoError(t, err)
	assert.Greater(t, len(buf), 1)
}

func TestProtobufEncoder_EncodeTracesError(t *testing.T) {
	encoder := protobufEncoder{}
	buf, err := encoder.EncodeTraces(nil)
	assert.Nil(t, buf)
	assert.Error(t, err)
}

func TestNewProtobufTracesUnmarshaler(t *testing.T) {
	m := NewProtobufTracesUnmarshaler(false, false)
	assert.NotNil(t, m)
	assert.Implements(t, (*pdata.TracesUnmarshaler)(nil), m)
}

func TestNewProtobufTracesMarshaler(t *testing.T) {
	m := NewProtobufTracesMarshaler()
	assert.NotNil(t, m)
	assert.Implements(t, (*pdata.TracesMarshaler)(nil), m)
}
