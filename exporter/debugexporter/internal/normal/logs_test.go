// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestMarshalLogs(t *testing.T) {
	tests := []struct {
		name     string
		input    plog.Logs
		expected string
	}{
		{
			name:     "empty logs",
			input:    plog.NewLogs(),
			expected: "",
		},
		{
			name: "one log record",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				logRecord := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2024, 1, 23, 17, 54, 41, 153, time.UTC)))
				logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
				logRecord.SetSeverityText("INFO")
				logRecord.Body().SetStr("Single line log message")
				logRecord.Attributes().PutStr("key1", "value1")
				logRecord.Attributes().PutStr("key2", "value2")
				return logs
			}(),
			expected: `Single line log message key1=value1 key2=value2
`,
		},
		{
			name: "multiline log",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				logRecord := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2024, 1, 23, 17, 54, 41, 153, time.UTC)))
				logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
				logRecord.SetSeverityText("INFO")
				logRecord.Body().SetStr("First line of the log message\n  second line of the log message")
				logRecord.Attributes().PutStr("key1", "value1")
				logRecord.Attributes().PutStr("key2", "value2")
				return logs
			}(),
			expected: `First line of the log message
  second line of the log message key1=value1 key2=value2
`,
		},
		{
			name: "two log records",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				logRecords := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()

				logRecord := logRecords.AppendEmpty()
				logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2024, 1, 23, 17, 54, 41, 153, time.UTC)))
				logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
				logRecord.SetSeverityText("INFO")
				logRecord.Body().SetStr("Single line log message")
				logRecord.Attributes().PutStr("key1", "value1")
				logRecord.Attributes().PutStr("key2", "value2")

				logRecord = logRecords.AppendEmpty()
				logRecord.Body().SetStr("Multi-line\nlog message")
				logRecord.Attributes().PutStr("mykey2", "myvalue2")
				logRecord.Attributes().PutStr("mykey1", "myvalue1")
				return logs
			}(),
			expected: `Single line log message key1=value1 key2=value2
Multi-line
log message mykey2=myvalue2 mykey1=myvalue1
`,
		},
		{
			name: "log with maps in body and attributes",
			input: func() plog.Logs {
				logs := plog.NewLogs()
				logRecord := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)))
				logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
				logRecord.SetSeverityText("INFO")
				body := logRecord.Body().SetEmptyMap()
				body.PutStr("app", "CurrencyConverter")
				bodyEvent := body.PutEmptyMap("event")
				bodyEvent.PutStr("operation", "convert")
				bodyEvent.PutStr("result", "success")
				conversionAttr := logRecord.Attributes().PutEmptyMap("conversion")
				conversionSourceAttr := conversionAttr.PutEmptyMap("source")
				conversionSourceAttr.PutStr("currency", "USD")
				conversionSourceAttr.PutDouble("amount", 34.22)
				conversionDestinationAttr := conversionAttr.PutEmptyMap("destination")
				conversionDestinationAttr.PutStr("currency", "EUR")
				logRecord.Attributes().PutStr("service", "payments")
				return logs
			}(),
			expected: `{"app":"CurrencyConverter","event":{"operation":"convert","result":"success"}} conversion={"destination":{"currency":"EUR"},"source":{"amount":34.22,"currency":"USD"}} service=payments
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := NewNormalLogsMarshaler().MarshalLogs(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}
