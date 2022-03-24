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

package otlpgrpc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ json.Unmarshaler = TracesResponse{}
var _ json.Marshaler = TracesResponse{}

var _ json.Unmarshaler = TracesRequest{}
var _ json.Marshaler = TracesRequest{}

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

func TestTracesRequestJSON(t *testing.T) {
	mr := NewTracesRequest()
	assert.NoError(t, mr.UnmarshalJSON(tracesRequestJSON))
	assert.Equal(t, "test_span", mr.Traces().ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Name())

	got, err := mr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(tracesRequestJSON)), ""), string(got))
}
