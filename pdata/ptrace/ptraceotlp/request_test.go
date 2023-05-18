// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
