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

package ptraceotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ json.Unmarshaler = ExportRequest{}
var _ json.Marshaler = ExportRequest{}

var tracesRequestJSON = []byte(`
	{
		"resourceSpans": [
			{
				"resource": {},
				"scopeSpans": [
					{
						"scope": {},
						"spans": [
							{
								"traceId": "",
								"spanId":"",
								"parentSpanId":"",
								"name": "test_span",
								"status": {}
							}
						]
					}
				]
			}
		]
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewExportRequest()
	assert.Equal(t, tr.Traces().SpanCount(), 0)
	tr.Traces().ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	assert.Equal(t, tr.Traces().SpanCount(), 1)
}

func TestRequestJSON(t *testing.T) {
	tr := NewExportRequest()
	assert.NoError(t, tr.UnmarshalJSON(tracesRequestJSON))
	assert.Equal(t, "test_span", tr.Traces().ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())

	got, err := tr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(tracesRequestJSON)), ""), string(got))
}
