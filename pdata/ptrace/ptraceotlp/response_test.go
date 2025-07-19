// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptraceotlp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ json.Unmarshaler = ExportResponse{}
	_ json.Marshaler   = ExportResponse{}
)

func TestExportResponseJSON(t *testing.T) {
	jsonStr := `{"partialSuccess": {"rejectedSpans":1, "errorMessage":"nothing"}}`
	val := NewExportResponse()
	require.NoError(t, val.UnmarshalJSON([]byte(jsonStr)))
	expected := NewExportResponse()
	expected.PartialSuccess().SetRejectedSpans(1)
	expected.PartialSuccess().SetErrorMessage("nothing")
	assert.Equal(t, expected, val)
}

func TestUnmarshalJSONExportResponse(t *testing.T) {
	jsonStr := `{"extra":"", "partialSuccess": {}}`
	val := NewExportResponse()
	require.NoError(t, val.UnmarshalJSON([]byte(jsonStr)))
	assert.Equal(t, NewExportResponse(), val)
}
