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

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextLogsMarshaler().MarshalLogs(tt.in)
			assert.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "logs", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}
