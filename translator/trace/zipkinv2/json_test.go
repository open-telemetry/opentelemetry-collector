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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestJSONDecoder_DecodeTraces(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/zipkin_v2_single.json")
	require.NoError(t, err)
	decoder := jsonDecoder{}
	spans, err := decoder.DecodeTraces(data)
	assert.NoError(t, err)
	assert.NotNil(t, spans)
	assert.IsType(t, []*zipkinmodel.SpanModel{}, spans)
}

func TestJSONDecoder_DecodeTracesError(t *testing.T) {
	decoder := jsonDecoder{}
	spans, err := decoder.DecodeTraces([]byte("{"))
	assert.Error(t, err)
	assert.Nil(t, spans)
}

func TestJSONEncoder_EncodeTraces(t *testing.T) {
	encoder := jsonEncoder{}
	buf, err := encoder.EncodeTraces(generateSpanErrorTags())
	assert.NoError(t, err)
	assert.Greater(t, len(buf), 1)
}

func TestJSONEncoder_EncodeTracesError(t *testing.T) {
	encoder := jsonEncoder{}
	buf, err := encoder.EncodeTraces(nil)
	assert.Error(t, err)
	assert.Nil(t, buf)
}

func TestNewJSONTracesUnmarshaler(t *testing.T) {
	m := NewJSONTracesUnmarshaler(false)
	assert.NotNil(t, m)
	assert.Implements(t, (*pdata.TracesUnmarshaler)(nil), m)
}

func TestNewJSONTracesMarshaler(t *testing.T) {
	m := NewJSONTracesMarshaler()
	assert.NotNil(t, m)
	assert.Implements(t, (*pdata.TracesMarshaler)(nil), m)
}
