// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	"go.opentelemetry.io/collector/pdata/plog"
)

type LogsSizer interface {
	LogsSize(ld plog.Logs) int
	ResourceLogsSize(rl plog.ResourceLogs) int
	ScopeLogsSize(sl plog.ScopeLogs) int
	LogRecordSize(lr plog.LogRecord) int

	// DeltaSize returns the delta size when a ResourceLog, ScopeLog or LogRecord is added.
	DeltaSize(newItemSize int) int
}

// LogsBytesSizer returns the byte size of serialized protos.
type LogsBytesSizer struct {
	plog.ProtoMarshaler
	protoDeltaSizer
}

// LogsCountSizer returns the nunmber of logs entries.
type LogsCountSizer struct{}

func (s *LogsCountSizer) LogsSize(ld plog.Logs) int {
	return ld.LogRecordCount()
}

func (s *LogsCountSizer) ResourceLogsSize(rl plog.ResourceLogs) int {
	count := 0
	for k := 0; k < rl.ScopeLogs().Len(); k++ {
		count += rl.ScopeLogs().At(k).LogRecords().Len()
	}
	return count
}

func (s *LogsCountSizer) ScopeLogsSize(sl plog.ScopeLogs) int {
	return sl.LogRecords().Len()
}

func (s *LogsCountSizer) LogRecordSize(_ plog.LogRecord) int {
	return 1
}

func (s *LogsCountSizer) DeltaSize(newItemSize int) int {
	return newItemSize
}
