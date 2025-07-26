// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogotlp

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
							}
						}
					]
				}
			]
		}
		]
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewExportRequest()
	assert.Equal(t, 0, tr.Logs().LogRecordCount())
	tr.Logs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	assert.Equal(t, 1, tr.Logs().LogRecordCount())
}

func TestRequestJSON(t *testing.T) {
	lr := NewExportRequest()
	require.NoError(t, lr.UnmarshalJSON(logsRequestJSON))
	assert.Equal(t, "test_log_record", lr.Logs().ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().AsString())

	got, err := lr.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(logsRequestJSON)), ""), string(got))
}
