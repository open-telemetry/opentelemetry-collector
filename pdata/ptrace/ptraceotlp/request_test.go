// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptraceotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ json.Unmarshaler = ExportRequest{}
	_ json.Marshaler   = ExportRequest{}
)

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
	assert.Equal(t, 0, tr.Traces().SpanCount())
	tr.Traces().ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	assert.Equal(t, 1, tr.Traces().SpanCount())
}

func TestRequestJSON(t *testing.T) {
	tr := NewExportRequest()
	require.NoError(t, tr.UnmarshalJSON(tracesRequestJSON))
	assert.Equal(t, "test_span", tr.Traces().ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())

	got, err := tr.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(tracesRequestJSON)), ""), string(got))
}
