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

	"go.opentelemetry.io/collector/exporter/debugexporter/internal"
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
			got, err := NewTextLogsMarshaler(internal.NewDefaultOutputConfig()).MarshalLogs(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "logs", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

func TestLogsWithOutputConfig(t *testing.T) {
	tests := []struct {
		name   string
		in     plog.Logs
		out    string
		config internal.OutputConfig
	}{
		{
			name: "marshal_logs_with_attributes_filter_include",
			in:   generateBasicLogs(),
			out:  "log_with_attributes_filter.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep", "app", "instance_num"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_logs_with_attributes_filter_exclude",
			in:   generateBasicLogs(),
			out:  "log_with_attributes_filter.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Exclude: []string{"attribute.remove"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Exclude: []string{"attribute.remove"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Exclude: []string{"attribute.remove"},
					},
				},
			},
		},
		{
			name: "marshal_logs_with_record_disabled",
			in:   generateBasicLogs(),
			out:  "log_with_attributes_filter_with_record_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep", "app", "instance_num"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_logs_with_scope_disabled",
			in:   generateBasicLogs(),
			out:  "log_with_attributes_filter_with_scope_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep", "app", "instance_num"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_logs_with_resource_disabled",
			in:   generateBasicLogs(),
			out:  "log_with_attributes_filter_with_resource_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep", "app", "instance_num"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: false,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: true,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
		{
			name: "marshal_logs_with_attributes_disabled",
			in:   generateBasicLogs(),
			out:  "log_with_attributes_filter_with_attributes_disabled.out",
			config: internal.OutputConfig{
				Scope: internal.ScopeOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: false,
						Include: []string{"attribute.keep"},
					},
				},
				Record: internal.RecordOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: false,
						Include: []string{"attribute.keep", "app", "instance_num"},
					},
				},
				Resource: internal.ResourceOutputConfig{
					Enabled: true,
					AttributesOutputConfig: internal.AttributesOutputConfig{
						Enabled: false,
						Include: []string{"attribute.keep"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextLogsMarshaler(tt.config).MarshalLogs(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "logs", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

func generateBasicLogs() plog.Logs {
	ls := plog.NewLogs()
	rl := ls.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("attribute.keep", "resource-keep")
	rl.Resource().Attributes().PutStr("attribute.remove", "resource-remove")
	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().Attributes().PutStr("attribute.keep", "scope-keep")
	sl.Scope().Attributes().PutStr("attribute.remove", "scope-remove")
	l := sl.LogRecords().AppendEmpty()
	l.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)))
	l.SetSeverityNumber(plog.SeverityNumberInfo)
	l.SetSeverityText("Info")
	l.SetEventName("event_name")
	l.Body().SetStr("This is a log message")
	attrs := l.Attributes()
	attrs.PutStr("app", "server")
	attrs.PutInt("instance_num", 1)
	attrs.PutStr("attribute.keep", "logRecord-keep")
	attrs.PutStr("attribute.remove", "logRecord-remove")
	l.SetSpanID([8]byte{0x01, 0x02, 0x04, 0x08})
	l.SetTraceID([16]byte{0x08, 0x04, 0x02, 0x01})
	return ls
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
