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

package plog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProtoLogsUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalLogs([]byte("+$%"))
	assert.Error(t, err)
}

func TestLogsProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	ld := NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().SetSeverityText("error")

	size := marshaler.LogsSize(ld)

	bytes, err := marshaler.MarshalLogs(ld)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)

}

func TestResourceLogsProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	rl := NewResourceLogs()
	rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().SetSeverityText("error")

	size := marshaler.ResourceLogsSize(rl)

	bytes, err := marshaler.MarshalResourceLogs(rl)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)

}

func TestLogRecordProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	lr := NewLogRecord()
	lr.SetSeverityText("error")

	size := marshaler.LogRecordSize(lr)

	bytes, err := marshaler.MarshalLogRecord(lr)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)

}

func TestProtoSizerEmptyLogs(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 0, sizer.LogsSize(NewLogs()))
	assert.Equal(t, 2, sizer.ResourceLogsSize(NewResourceLogs())) // 2 due to non-nil values
	assert.Equal(t, 6, sizer.LogRecordSize(NewLogRecord()))       // 6 due to non-nil values
}

func BenchmarkLogsToProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	logs := generateBenchmarkLogs(128)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalLogs(logs)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkLogsFromProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseLogs := generateBenchmarkLogs(128)
	buf, err := marshaler.MarshalLogs(baseLogs)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logs, err := unmarshaler.UnmarshalLogs(buf)
		require.NoError(b, err)
		assert.Equal(b, baseLogs.ResourceLogs().Len(), logs.ResourceLogs().Len())
	}
}

func BenchmarkResourceLogsToProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	resourceLogs := generateBenchmarkResourceLogs(128)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalResourceLogs(resourceLogs)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkResourceLogsFromProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseResourceLogs := generateBenchmarkResourceLogs(128)
	buf, err := marshaler.MarshalResourceLogs(baseResourceLogs)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		resourceLogs, err := unmarshaler.UnmarshalResourceLogs(buf)
		require.NoError(b, err)
		assert.Equal(b, baseResourceLogs.ScopeLogs().Len(), resourceLogs.ScopeLogs().Len())
	}
}

func BenchmarkLogRecordToProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	logRecord := generateBenchmarkLogRecord()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalLogRecord(logRecord)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkLogRecordFromProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseLogRecord := generateBenchmarkLogRecord()
	buf, err := marshaler.MarshalLogRecord(baseLogRecord)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logRecord, err := unmarshaler.UnmarshalLogRecord(buf)
		require.NoError(b, err)
		assert.Equal(b, baseLogRecord.Body().Bytes().Len(), logRecord.Body().Bytes().Len())
	}
}

func generateBenchmarkLogs(logsCount int) Logs {
	endTime := pcommon.NewTimestampFromTime(time.Now())

	md := NewLogs()
	ilm := md.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	ilm.LogRecords().EnsureCapacity(logsCount)
	for i := 0; i < logsCount; i++ {
		im := ilm.LogRecords().AppendEmpty()
		im.SetTimestamp(endTime)
	}
	return md
}

func generateBenchmarkResourceLogs(logsCount int) ResourceLogs {
	endTime := pcommon.NewTimestampFromTime(time.Now())

	md := NewResourceLogs()
	ilm := md.ScopeLogs().AppendEmpty()
	ilm.LogRecords().EnsureCapacity(logsCount)
	for i := 0; i < logsCount; i++ {
		im := ilm.LogRecords().AppendEmpty()
		im.SetTimestamp(endTime)
	}
	return md
}

func generateBenchmarkLogRecord() LogRecord {
	timestamp := pcommon.NewTimestampFromTime(time.Now())

	md := NewLogRecord()
	md.SetTimestamp(timestamp)
	return md
}
