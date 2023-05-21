// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogotlp

import (
	"encoding/json"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

var _ json.Unmarshaler = ExportResponse{}
var _ json.Marshaler = ExportResponse{}

func TestExportResponseJSON(t *testing.T) {
	jsonStr := `{"partialSuccess": {"rejectedLogRecords":1, "errorMessage":"nothing"}}`
	val := NewExportResponse()
	assert.NoError(t, val.UnmarshalJSON([]byte(jsonStr)))
	expected := NewExportResponse()
	expected.PartialSuccess().SetRejectedLogRecords(1)
	expected.PartialSuccess().SetErrorMessage("nothing")
	assert.Equal(t, expected, val)
}

func TestUnmarshalJSONExportResponse(t *testing.T) {
	jsonStr := `{"extra":"", "partialSuccess": {}}`
	val := NewExportResponse()
	assert.NoError(t, val.UnmarshalJSON([]byte(jsonStr)))
	assert.Equal(t, NewExportResponse(), val)
}

func TestUnmarshalJsoniterExportPartialSuccess(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewExportPartialSuccess()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewExportPartialSuccess(), val)
}
