// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestLogsText(t *testing.T) {
	tests := []struct {
		name string
		in   plog.Logs
		out  string
	}{
		{
			name: "empty_logs",
			in:   plog.NewLogs(),
			out:  "empty.out",
		},
		{
			name: "logs_with_one_record",
			in:   testdata.GenerateLogs(1),
			out:  "one_record.out",
		},
		{
			name: "logs_with_two_records",
			in:   testdata.GenerateLogs(2),
			out:  "two_records.out",
		},
		{
			name: "log_with_event_name",
			in: func() plog.Logs {
				ls := plog.NewLogs()
				l := ls.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				l.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)))
				l.SetSeverityNumber(plog.SeverityNumberInfo)
				l.SetSeverityText("Info")
				l.SetEventName("event_name")
				l.Body().SetStr("This is a log message")
				attrs := l.Attributes()
				attrs.PutStr("app", "server")
				attrs.PutInt("instance_num", 1)
				l.SetSpanID([8]byte{0x01, 0x02, 0x04, 0x08})
				l.SetTraceID([16]byte{0x08, 0x04, 0x02, 0x01})
				return ls
			}(),
			out: "log_with_event_name.out",
		},
		{
			name: "logs_with_embedded_maps",
			in: func() plog.Logs {
				ls := plog.NewLogs()
				l := ls.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				l.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)))
				l.SetSeverityNumber(plog.SeverityNumberInfo)
				l.SetSeverityText("INFO")
				bm := l.Body().SetEmptyMap()
				bm.PutStr("key1", "val1")
				bmm := bm.PutEmptyMap("key2")
				bmm.PutStr("key21", "val21")
				bmm.PutStr("key22", "val22")
				am := l.Attributes().PutEmptyMap("key1")
				am.PutStr("key11", "val11")
				am.PutStr("key12", "val12")
				am.PutEmptyMap("key13").PutStr("key131", "val131")
				l.Attributes().PutStr("key2", "val2")
				return ls
			}(),
			out: "embedded_maps.out",
		},
		{
			name: "logs_with_entity_refs",
			in:   generateLogsWithEntityRefs(),
			out:  "logs_with_entity_refs.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextLogsMarshaler().MarshalLogs(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "logs", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

// TestLogRecordTraceSpanIDs specifically tests that TraceID and SpanID are rendered correctly
// when present in log records, ensuring proper formatting that matches trace output.
func TestLogRecordTraceSpanIDs(t *testing.T) {
	ls := plog.NewLogs()
	lr := ls.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	
	// Set non-empty TraceID and SpanID
	traceID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	lr.SetTraceID(traceID)
	lr.SetSpanID(spanID)
	
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 0, time.UTC)))
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.Body().SetStr("Log with trace context")
	
	got, err := NewTextLogsMarshaler().MarshalLogs(ls)
	require.NoError(t, err)
	
	output := string(got)
	
	// Verify TraceID is present and properly formatted with indentation
	assert.Contains(t, output, "    Trace ID       : 0102030405060708090a0b0c0d0e0f10")
	
	// Verify SpanID is present and properly formatted with indentation
	assert.Contains(t, output, "    Span ID        : 1112131415161718")
	
	// Ensure the output uses lowercase hex as per OTLP convention
	assert.NotContains(t, output, "0102030405060708090A0B0C0D0E0F10") // uppercase should not appear
}

func generateLogsWithEntityRefs() plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()

	setupResourceWithEntityRefs(rl.Resource())

	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("test-scope")
	lr := sl.LogRecords().AppendEmpty()
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)))
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.SetSeverityText("Info")
	lr.Body().SetStr("This is a test log message")
	lr.Attributes().PutStr("test.attribute", "test-value")

	return ld
}
