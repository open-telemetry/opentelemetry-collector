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

package ptrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var tracesOTLP = func() Traces {
	td := NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	il := rs.ScopeSpans().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Spans().AppendEmpty().SetName("testSpan")
	return td
}()

var tracesJSON = `{"resourceSpans":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"scopeSpans":[{"scope":{"name":"name","version":"version"},"spans":[{"traceId":"","spanId":"","parentSpanId":"","name":"testSpan","status":{}}]}]}]}`

func TestTracesJSON(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)

	decoder := NewJSONUnmarshaler()
	var got interface{}
	got, err = decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, tracesOTLP, got)
}

func TestTracesJSON_Marshal(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)
	assert.Equal(t, tracesJSON, string(jsonBuf))
}
