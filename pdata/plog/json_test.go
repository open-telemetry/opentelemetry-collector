// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

var _ Marshaler = (*JSONMarshaler)(nil)
var _ Unmarshaler = (*JSONUnmarshaler)(nil)

var logsOTLP = func() Logs {
	ld := NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("host.name", "testHost")
	rl.Resource().SetDroppedAttributesCount(1)
	rl.SetSchemaUrl("resource_schema")
	il := rl.ScopeLogs().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Scope().SetDroppedAttributesCount(1)
	il.SetSchemaUrl("scope_schema")
	lg := il.LogRecords().AppendEmpty()
	lg.SetSeverityNumber(SeverityNumber(otlplogs.SeverityNumber_SEVERITY_NUMBER_ERROR))
	lg.SetSeverityText("Error")
	lg.SetDroppedAttributesCount(1)
	lg.SetFlags(LogRecordFlags(otlplogs.LogRecordFlags_LOG_RECORD_FLAGS_DO_NOT_USE))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	lg.SetTraceID(traceID)
	lg.SetSpanID(spanID)
	lg.Body().SetStr("hello world")
	lg.SetTimestamp(pcommon.Timestamp(1684617382541971000))
	lg.SetObservedTimestamp(pcommon.Timestamp(1684623646539558000))
	lg.Attributes().PutStr("sdkVersion", "1.0.1")
	lg.SetFlags(DefaultLogRecordFlags.WithIsSampled(true))
	return ld
}()

func TestLogsJSON(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalLogs(logsOTLP)
	assert.NoError(t, err)
	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalLogs(jsonBuf)
	assert.NoError(t, err)
	assert.EqualValues(t, logsOTLP, got)
}

var logsJSON = `{"resourceLogs":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}],"droppedAttributesCount":1},"scopeLogs":[{"scope":{"name":"name","version":"version","droppedAttributesCount":1},"logRecords":[{"timeUnixNano":"1684617382541971000","observedTimeUnixNano":"1684623646539558000","severityNumber":17,"severityText":"Error","body":{"stringValue":"hello world"},"attributes":[{"key":"sdkVersion","value":{"stringValue":"1.0.1"}}],"droppedAttributesCount":1,"flags":1,"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1112131415161718"}],"schemaUrl":"scope_schema"}],"schemaUrl":"resource_schema"}]}`

func TestJSONUnmarshal(t *testing.T) {
	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalLogs([]byte(logsJSON))
	assert.NoError(t, err)
	assert.EqualValues(t, logsOTLP, got)
}

func TestJSONMarshal(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalLogs(logsOTLP)
	assert.NoError(t, err)
	assert.Equal(t, logsJSON, string(jsonBuf))
}

func TestJSONUnmarshalInvalid(t *testing.T) {
	jsonStr := `{"extra":"", "resourceLogs": "extra"}`
	decoder := &JSONUnmarshaler{}
	_, err := decoder.UnmarshalLogs([]byte(jsonStr))
	assert.Error(t, err)
}

func TestUnmarshalJsoniterLogsData(t *testing.T) {
	jsonStr := `{"extra":"", "resourceLogs": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewLogs()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewLogs(), val)
}

func TestUnmarshalJsoniterResourceLogs(t *testing.T) {
	jsonStr := `{"extra":"", "resource": {}, "scopeLogs": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewResourceLogs()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewResourceLogs(), val)
}

func TestUnmarshalJsoniterScopeLogs(t *testing.T) {
	jsonStr := `{"extra":"", "scope": {}, "logRecords": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewScopeLogs()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewScopeLogs(), val)
}

func TestUnmarshalJsoniterLogRecord(t *testing.T) {
	jsonStr := `{"extra":"", "body":{}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewLogRecord()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewLogRecord(), val)
}

func TestUnmarshalJsoniterLogWrongTraceID(t *testing.T) {
	jsonStr := `{"body":{}, "traceId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewLogRecord().unmarshalJsoniter(iter)
	require.Error(t, iter.Error)
	assert.Contains(t, iter.Error.Error(), "parse trace_id")
}

func TestUnmarshalJsoniterLogWrongSpanID(t *testing.T) {
	jsonStr := `{"body":{}, "spanId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewLogRecord().unmarshalJsoniter(iter)
	require.Error(t, iter.Error)
	assert.Contains(t, iter.Error.Error(), "parse span_id")
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()

	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalLogs(logsOTLP)
	assert.NoError(b, err)
	decoder := &JSONUnmarshaler{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := decoder.UnmarshalLogs(jsonBuf)
			assert.NoError(b, err)
		}
	})
}
