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

var _ json.Unmarshaler = LogsResponse{}
var _ json.Marshaler = LogsResponse{}

var _ json.Unmarshaler = LogsRequest{}
var _ json.Marshaler = LogsRequest{}

var logsRequestJSON = []byte(`
	{
	  "resourceLogs": [
		{
          "resource": {},
		  "scopeLogs": [
			{
              "scope": {},
			  "logRecords": [
				{
				  "body": {
	                "stringValue": "test_log_record"
                  },
				  "traceId": "",
				  "spanId": ""
				}
			  ]
			}
		  ]
		}
	  ]
	}`)

func TestLogsRequestJSON(t *testing.T) {
	lr := NewLogsRequest()
	assert.NoError(t, lr.UnmarshalJSON(logsRequestJSON))
	assert.Equal(t, "test_log_record", lr.Logs().ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).LogRecords().At(0).Body().AsString())

	got, err := lr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(logsRequestJSON)), ""), string(got))
}
