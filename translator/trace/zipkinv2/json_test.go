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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestJSONUnmarshaler_UnmarshalTraces(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/zipkin_v2_single.json")
	require.NoError(t, err)
	decoder := NewJSONTracesUnmarshaler(false)
	td, err := decoder.UnmarshalTraces(data)
	assert.NoError(t, err)
	assert.Equal(t, 1, td.SpanCount())
}

func TestJSONUnmarshaler_DecodeTracesError(t *testing.T) {
	decoder := NewJSONTracesUnmarshaler(false)
	_, err := decoder.UnmarshalTraces([]byte("{"))
	assert.Error(t, err)
}

func TestJSONEncoder_EncodeTraces(t *testing.T) {
	marshaler := NewJSONTracesMarshaler()
	buf, err := marshaler.MarshalTraces(generateTraceSingleSpanErrorStatus())
	assert.NoError(t, err)
	assert.Greater(t, len(buf), 1)
}

func TestJSONEncoder_EncodeTracesError(t *testing.T) {
	invalidTD := pdata.NewTraces()
	// Add one span with empty trace ID.
	invalidTD.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	marshaler := NewJSONTracesMarshaler()
	buf, err := marshaler.MarshalTraces(invalidTD)
	assert.Error(t, err)
	assert.Nil(t, buf)
}
