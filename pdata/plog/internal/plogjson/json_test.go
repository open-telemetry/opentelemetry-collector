// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogjson

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

func TestReadLogsData(t *testing.T) {
	jsonStr := `{"extra":"", "resourceLogs": [{"extra":""}]}`
	value := &otlplogs.LogsData{}
	assert.NoError(t, UnmarshalLogsData([]byte(jsonStr), value))
	assert.Equal(t, &otlplogs.LogsData{ResourceLogs: []*otlplogs.ResourceLogs{{}}}, value)
}

func TestReadExportLogsServiceRequest(t *testing.T) {
	jsonStr := `{"extra":"", "resourceLogs": [{"extra":""}]}`
	value := &otlpcollectorlog.ExportLogsServiceRequest{}
	assert.NoError(t, UnmarshalExportLogsServiceRequest([]byte(jsonStr), value))
	assert.Equal(t, &otlpcollectorlog.ExportLogsServiceRequest{ResourceLogs: []*otlplogs.ResourceLogs{{}}}, value)
}

func TestReadExportLogsServiceResponse(t *testing.T) {
	jsonStr := `{"extra":"", "partialSuccess": {}}`
	value := &otlpcollectorlog.ExportLogsServiceResponse{}
	assert.NoError(t, UnmarshalExportLogsServiceResponse([]byte(jsonStr), value))
	assert.Equal(t, &otlpcollectorlog.ExportLogsServiceResponse{}, value)
}

func TestReadResourceLogs(t *testing.T) {
	jsonStr := `{"extra":"", "resource": {}, "schemaUrl": "schema", "scopeLogs": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readResourceLogs(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlplogs.ResourceLogs{SchemaUrl: "schema"}, value)
}

func TestReadScopeLogs(t *testing.T) {
	jsonStr := `{"extra":"", "scope": {}, "logRecords": [], "schemaUrl": "schema"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readScopeLogs(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlplogs.ScopeLogs{SchemaUrl: "schema"}, value)
}

func TestReadLogWrongTraceID(t *testing.T) {
	jsonStr := `{"severityText":"Error", "body":{}, "traceId":"--", "spanId":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readLog(iter)
	require.Error(t, iter.Error)
	assert.Contains(t, iter.Error.Error(), "parse trace_id")
}

func TestReadLogWrongSpanID(t *testing.T) {
	jsonStr := `{"severityText":"Error", "body":{}, "traceId":"", "spanId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readLog(iter)
	require.Error(t, iter.Error)
	assert.Contains(t, iter.Error.Error(), "parse span_id")
}

func TestReadExportLogsPartialSuccess(t *testing.T) {
	jsonStr := `{"extra":"", "rejectedLogRecords":1, "errorMessage":"nothing"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readExportLogsPartialSuccess(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, otlpcollectorlog.ExportLogsPartialSuccess{RejectedLogRecords: 1, ErrorMessage: "nothing"}, value)
}
