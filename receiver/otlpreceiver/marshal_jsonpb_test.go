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

package otlpreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	v1 "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/internal/testdata"
)

const expectedJSON = `{
  "resource": {
    "attributes": [
      {
        "key": "resource-attr",
        "value": {
          "stringValue": "resource-attr-val-1"
        }
      }
    ]
  },
  "instrumentationLibrarySpans": [
    {
      "instrumentationLibrary": {},
      "spans": [
        {
          "traceId": "",
          "spanId": "",
          "parentSpanId": "",
          "name": "operationA",
          "startTimeUnixNano": "1581452772000000321",
          "endTimeUnixNano": "1581452773000000789",
          "droppedAttributesCount": 1,
          "events": [
            {
              "timeUnixNano": "1581452773000000123",
              "name": "event-with-attr",
              "attributes": [
                {
                  "key": "span-event-attr",
                  "value": {
                    "stringValue": "span-event-attr-val"
                  }
                }
              ],
              "droppedAttributesCount": 2
            },
            {
              "timeUnixNano": "1581452773000000123",
              "name": "event",
              "droppedAttributesCount": 2
            }
          ],
          "droppedEventsCount": 1,
          "status": {
            "deprecatedCode": "DEPRECATED_STATUS_CODE_UNKNOWN_ERROR",
            "message": "status-cancelled",
            "code": "STATUS_CODE_ERROR"
          }
        }
      ]
    }
  ]
}`

func TestJSONPbMarshal(t *testing.T) {
	jpb := JSONPb{
		Indent: "  ",
	}
	td := testdata.GenerateTraceDataOneSpan()
	otlp := pdata.TracesToOtlp(td)
	bytes, err := jpb.Marshal(otlp[0])
	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, string(bytes))
}

func TestJSONPbUnmarshal(t *testing.T) {
	jpb := JSONPb{
		Indent: "  ",
	}
	var proto v1.ResourceSpans
	err := jpb.Unmarshal([]byte(expectedJSON), &proto)
	assert.NoError(t, err)
	td := testdata.GenerateTraceDataOneSpan()
	otlp := pdata.TracesToOtlp(td)
	assert.EqualValues(t, &proto, otlp[0])
}
