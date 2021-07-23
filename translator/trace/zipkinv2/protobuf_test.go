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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestProtobufMarshalUnmarshal(t *testing.T) {
	pb, err := NewProtobufTracesMarshaler().MarshalTraces(generateTraceSingleSpanErrorStatus())
	require.NoError(t, err)

	pbUnmarshaler := protobufUnmarshaler{debugWasSet: false}
	td, err := pbUnmarshaler.UnmarshalTraces(pb)
	assert.NoError(t, err)
	assert.Equal(t, generateTraceSingleSpanErrorStatus(), td)
}

func TestProtobuf_UnmarshalTracesError(t *testing.T) {
	decoder := protobufUnmarshaler{debugWasSet: false}
	_, err := decoder.UnmarshalTraces([]byte("{"))
	assert.Error(t, err)
}

func TestProtobuf_MarshalTracesError(t *testing.T) {
	invalidTD := pdata.NewTraces()
	// Add one span with empty trace ID.
	invalidTD.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	marshaler := NewProtobufTracesMarshaler()
	buf, err := marshaler.MarshalTraces(invalidTD)
	assert.Nil(t, buf)
	assert.Error(t, err)
}
