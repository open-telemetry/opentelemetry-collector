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

package jsonstream

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestLogsJSON(t *testing.T) {
	testCases := []struct {
		name   string
		logs   pdata.Logs
		expect string
	}{
		{
			"NewLogs",
			pdata.NewLogs(),
			``,
		},
		{
			"GenerateLogsOneEmptyResourceLogs",
			testdata.GenerateLogsOneEmptyResourceLogs(),
			compactJSON(`{
			  "resource": {
			    "type": "resourceLog",
			    "labels": {}
			  }
			}`),
		},
		{
			"GenerateLogsNoLogRecords",
			testdata.GenerateLogsNoLogRecords(),
			compactJSON(`{
			  "resource": {
			    "type": "resourceLog",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  }
			}`),
		},
		{
			"GenerateLogsOneEmptyLogRecord",
			testdata.GenerateLogsOneEmptyLogRecord(),
			compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "1970-01-01 00:00:00 +0000 UTC",
			  "severityText": "",
			  "severityNumber": "SEVERITY_NUMBER_UNSPECIFIED",
			  "name": "",
			  "body": null,
			  "droppedAttributesCount": 0,
			  "attributes": {},
			  "traceID": "",
			  "spanID": "",
			  "flags": 0
			}`),
		},
		{
			"GenerateLogsOneLogRecord",
			testdata.GenerateLogsOneLogRecord(),
			compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "severityText": "Info",
			  "severityNumber": "SEVERITY_NUMBER_INFO",
			  "name": "logA",
			  "body": "This is a log message",
			  "droppedAttributesCount": 1,
			  "attributes": {
			    "app": "server",
			    "instance_num": 1
			  },
			  "traceID": "08040201000000000000000000000000",
			  "spanID": "0102040800000000",
			  "flags": 0
			}`),
		},
		{
			"GenerateLogsOneLogRecordNoResource",
			testdata.GenerateLogsOneLogRecordNoResource(),
			compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {}
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "severityText": "Info",
			  "severityNumber": "SEVERITY_NUMBER_INFO",
			  "name": "logA",
			  "body": "This is a log message",
			  "droppedAttributesCount": 1,
			  "attributes": {
			    "app": "server",
			    "instance_num": 1
			  },
			  "traceID": "08040201000000000000000000000000",
			  "spanID": "0102040800000000",
			  "flags": 0
			}`),
		},
		{
			"GenerateLogsTwoLogRecordsSameResource",
			testdata.GenerateLogsTwoLogRecordsSameResource(),
			lines(compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "severityText": "Info",
			  "severityNumber": "SEVERITY_NUMBER_INFO",
			  "name": "logA",
			  "body": "This is a log message",
			  "droppedAttributesCount": 1,
			  "attributes": {
			    "app": "server",
			    "instance_num": 1
			  },
			  "traceID": "08040201000000000000000000000000",
			  "spanID": "0102040800000000",
			  "flags": 0
			}`), compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "severityText": "Info",
			  "severityNumber": "SEVERITY_NUMBER_INFO",
			  "name": "logB",
			  "body": "something happened",
			  "droppedAttributesCount": 1,
			  "attributes": {
			    "customer": "acme",
			    "env": "dev"
			  },
			  "traceID": "",
			  "spanID": "",
			  "flags": 0
			}`)),
		},
		{
			"GenerateLogsTwoLogRecordsSameResourceOneDifferent",
			testdata.GenerateLogsTwoLogRecordsSameResourceOneDifferent(),
			lines(compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "severityText": "Info",
			  "severityNumber": "SEVERITY_NUMBER_INFO",
			  "name": "logA",
			  "body": "This is a log message",
			  "droppedAttributesCount": 1,
			  "attributes": {
			    "app": "server",
			    "instance_num": 1
			  },
			  "traceID": "08040201000000000000000000000000",
			  "spanID": "0102040800000000",
			  "flags": 0
			}`), compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "severityText": "Info",
			  "severityNumber": "SEVERITY_NUMBER_INFO",
			  "name": "logB",
			  "body": "something happened",
			  "droppedAttributesCount": 1,
			  "attributes": {
			    "customer": "acme",
			    "env": "dev"
			  },
			  "traceID": "",
			  "spanID": "",
			  "flags": 0
			}`), compactJSON(`{
			  "resource": {
			    "type": "log",
			    "labels": {
			      "resource-attr": "resource-attr-val-2"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "severityText": "Warning",
			  "severityNumber": "SEVERITY_NUMBER_WARN",
			  "name": "logC",
			  "body": "something else happened",
			  "droppedAttributesCount": 1,
			  "attributes": {},
			  "traceID": "",
			  "spanID": "",
			  "flags": 0
			}`)),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := NewJSONLogsMarshaler().MarshalLogs(tt.logs)
			if assert.NoError(t, err) {
				testJSON(t, tt.expect, string(logs))
			}
		})
	}
}
