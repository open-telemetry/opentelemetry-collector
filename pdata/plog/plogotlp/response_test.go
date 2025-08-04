// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogotlp

import (
	stdjson "encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

var (
	_ stdjson.Unmarshaler = ExportResponse{}
	_ stdjson.Marshaler   = ExportResponse{}
)

func TestExportResponseJSON(t *testing.T) {
	jsonStr := `{"partialSuccess": {"rejectedLogRecords":"1", "errorMessage":"nothing"}}`
	val := NewExportResponse()
	require.NoError(t, val.UnmarshalJSON([]byte(jsonStr)))
	expected := NewExportResponse()
	expected.PartialSuccess().SetRejectedLogRecords(1)
	expected.PartialSuccess().SetErrorMessage("nothing")
	assert.Equal(t, expected, val)
	buf, err := val.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, jsonStr, string(buf))
}

func TestUnmarshalJSONExportResponse(t *testing.T) {
	jsonStr := `{"extra":"", "partialSuccess": {}}`
	val := NewExportResponse()
	require.NoError(t, val.UnmarshalJSON([]byte(jsonStr)))
	assert.Equal(t, NewExportResponse(), val)
}

func TestUnmarshalJsoniterExportPartialSuccess(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)
	val := NewExportPartialSuccess()
	val.unmarshalJSONIter(iter)
	require.NoError(t, iter.Error())
	assert.Equal(t, NewExportPartialSuccess(), val)
}
