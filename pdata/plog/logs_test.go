// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestLogRecordCount(t *testing.T) {
	logs := NewLogs()
	assert.Equal(t, 0, logs.LogRecordCount())

	rl := logs.ResourceLogs().AppendEmpty()
	assert.Equal(t, 0, logs.LogRecordCount())

	ill := rl.ScopeLogs().AppendEmpty()
	assert.Equal(t, 0, logs.LogRecordCount())

	ill.LogRecords().AppendEmpty()
	assert.Equal(t, 1, logs.LogRecordCount())

	rms := logs.ResourceLogs()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeLogs().AppendEmpty()
	illl := rms.AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
	for range 5 {
		illl.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.Equal(t, 6, logs.LogRecordCount())
}

func TestLogRecordCountWithEmpty(t *testing.T) {
	assert.Zero(t, NewLogs().LogRecordCount())
	assert.Zero(t, newLogs(&internal.ExportLogsServiceRequest{
		ResourceLogs: []*internal.ResourceLogs{{}},
	}, new(internal.State)).LogRecordCount())
	assert.Zero(t, newLogs(&internal.ExportLogsServiceRequest{
		ResourceLogs: []*internal.ResourceLogs{
			{
				ScopeLogs: []*internal.ScopeLogs{{}},
			},
		},
	}, new(internal.State)).LogRecordCount())
	assert.Equal(t, 1, newLogs(&internal.ExportLogsServiceRequest{
		ResourceLogs: []*internal.ResourceLogs{
			{
				ScopeLogs: []*internal.ScopeLogs{
					{
						LogRecords: []*internal.LogRecord{{}},
					},
				},
			},
		},
	}, new(internal.State)).LogRecordCount())
}

func TestReadOnlyLogsInvalidUsage(t *testing.T) {
	ld := NewLogs()
	assert.False(t, ld.IsReadOnly())
	res := ld.ResourceLogs().AppendEmpty().Resource()
	res.Attributes().PutStr("k1", "v1")
	ld.MarkReadOnly()
	assert.True(t, ld.IsReadOnly())
	assert.Panics(t, func() { res.Attributes().PutStr("k2", "v2") })
}

func BenchmarkLogsUsage(b *testing.B) {
	ld := generateTestLogs()

	ts := pcommon.NewTimestampFromTime(time.Now())

	b.ReportAllocs()

	for b.Loop() {
		for i := 0; i < ld.ResourceLogs().Len(); i++ {
			rl := ld.ResourceLogs().At(i)
			res := rl.Resource()
			res.Attributes().PutStr("foo", "bar")
			v, ok := res.Attributes().Get("foo")
			assert.True(b, ok)
			assert.Equal(b, "bar", v.Str())
			v.SetStr("new-bar")
			assert.Equal(b, "new-bar", v.Str())
			res.Attributes().Remove("foo")
			for j := 0; j < rl.ScopeLogs().Len(); j++ {
				sl := rl.ScopeLogs().At(j)
				sl.Scope().SetName("new_test_name")
				assert.Equal(b, "new_test_name", sl.Scope().Name())
				for k := 0; k < sl.LogRecords().Len(); k++ {
					lr := sl.LogRecords().At(k)
					lr.Body().SetStr("new_body")
					assert.Equal(b, "new_body", lr.Body().Str())
					lr.SetTimestamp(ts)
					assert.Equal(b, ts, lr.Timestamp())
				}
				lr := sl.LogRecords().AppendEmpty()
				lr.Body().SetStr("another_log_record")
				lr.SetTimestamp(ts)
				lr.SetObservedTimestamp(ts)
				lr.SetSeverityText("info")
				lr.SetSeverityNumber(SeverityNumberInfo)
				lr.Attributes().PutStr("foo", "bar")
				lr.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
				sl.LogRecords().RemoveIf(func(lr LogRecord) bool {
					return lr.Body().Str() == "another_log_record"
				})
			}
		}
	}
}

func BenchmarkLogsMarshalJSON(b *testing.B) {
	ld := generateTestLogs()
	encoder := &JSONMarshaler{}

	b.ReportAllocs()

	for b.Loop() {
		jsonBuf, err := encoder.MarshalLogs(ld)
		require.NoError(b, err)
		require.NotNil(b, jsonBuf)
	}
}
