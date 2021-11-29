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

func TestTracesJSON(t *testing.T) {
	testCases := []struct {
		name   string
		traces pdata.Traces
		expect string
	}{
		{
			"NewTraces",
			pdata.NewTraces(),
			``,
		},
		{
			"GenerateTracesNoLibraries",
			testdata.GenerateTracesNoLibraries(),
			compactJSON(`{
			  "resource": {
			    "type": "resourceSpan",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  }
			}`),
		},
		{
			"GenerateTracesOneEmptyInstrumentationLibrary",
			testdata.GenerateTracesOneEmptyInstrumentationLibrary(),
			compactJSON(`{
			  "resource": {
			    "type": "instrumentationLibrarySpan",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  }
			}`),
		},
		{
			"GenerateTracesOneEmptyResourceSpans",
			testdata.GenerateTracesOneEmptyResourceSpans(),
			compactJSON(`{
			  "resource": {
			    "type": "resourceSpan",
			    "labels": {}
			  }
			}`),
		},
		{
			"GenerateTracesOneSpan",
			testdata.GenerateTracesOneSpan(),
			compactJSON(`{
			  "resource": {
			    "type": "span",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "traceID": "0102030405060708090a0b0c0d0e0f10",
			  "parentID": "",
			  "spanID": "1112131415161718",
			  "name": "operationA",
			  "kind": "SPAN_KIND_UNSPECIFIED",
			  "startTime": "2020-02-11 20:26:12.000000321 +0000 UTC",
			  "endTime": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "statusCode": "STATUS_CODE_ERROR",
			  "statusMessage": "status-cancelled",
			  "attributes": {},
			  "events": [
			    {
			      "name": "event-with-attr",
			      "timestamp": "2020-02-11 20:26:13.000000123 +0000 UTC",
			      "droppedAttributesCount": 2,
			      "attributes": {
			        "span-event-attr": "span-event-attr-val"
			      }
			    },
			    {
			      "name": "event",
			      "timestamp": "2020-02-11 20:26:13.000000123 +0000 UTC",
			      "droppedAttributesCount": 2,
			      "attributes": {}
			    }
			  ],
			  "links": []
			}`),
		},
		{
			"GenerateTracesTwoSpansSameResource",
			testdata.GenerateTracesTwoSpansSameResource(),
			lines(compactJSON(`{
			  "resource": {
			    "type": "span",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "traceID": "0102030405060708090a0b0c0d0e0f10",
			  "parentID": "",
			  "spanID": "1112131415161718",
			  "name": "operationA",
			  "kind": "SPAN_KIND_UNSPECIFIED",
			  "startTime": "2020-02-11 20:26:12.000000321 +0000 UTC",
			  "endTime": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "statusCode": "STATUS_CODE_ERROR",
			  "statusMessage": "status-cancelled",
			  "attributes": {},
			  "events": [
			    {
			      "name": "event-with-attr",
			      "timestamp": "2020-02-11 20:26:13.000000123 +0000 UTC",
			      "droppedAttributesCount": 2,
			      "attributes": {
			        "span-event-attr": "span-event-attr-val"
			      }
			    },
			    {
			      "name": "event",
			      "timestamp": "2020-02-11 20:26:13.000000123 +0000 UTC",
			      "droppedAttributesCount": 2,
			      "attributes": {}
			    }
			  ],
			  "links": []
			}`), compactJSON(`{
			  "resource": {
			    "type": "span",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "traceID": "",
			  "parentID": "",
			  "spanID": "",
			  "name": "operationB",
			  "kind": "SPAN_KIND_UNSPECIFIED",
			  "startTime": "2020-02-11 20:26:12.000000321 +0000 UTC",
			  "endTime": "2020-02-11 20:26:13.000000789 +0000 UTC",
			  "statusCode": "STATUS_CODE_UNSET",
			  "statusMessage": "",
			  "attributes": {},
			  "events": [],
			  "links": [
			    {
			      "traceID": "",
			      "linkID": "",
			      "traceState": "",
			      "droppedAttributesCount": 4,
			      "attributes": {
			        "span-link-attr": "span-link-attr-val"
			      }
			    },
			    {
			      "traceID": "",
			      "linkID": "",
			      "traceState": "",
			      "droppedAttributesCount": 4,
			      "attributes": {}
			    }
			  ]
			}`)),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			traces, err := NewJSONTracesMarshaler().MarshalTraces(tt.traces)
			if assert.NoError(t, err) {
				testJSON(t, tt.expect, string(traces))
			}
		})
	}
}
