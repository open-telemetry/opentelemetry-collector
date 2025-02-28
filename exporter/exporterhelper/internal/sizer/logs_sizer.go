// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	math_bits "math/bits"

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
}

// DeltaSize returns the delta size of a proto slice when a new item is added.
// Example:
//
//	prevSize := proto1.Size()
//	proto1.RepeatedField().AppendEmpty() = proto2
//
// Then currSize of proto1 can be calculated as
//
//	currSize := (prevSize + sizer.DeltaSize(proto2.Size()))
//
// This is derived from opentelemetry-collector/pdata/internal/data/protogen/logs/v1/logs.pb.go
// which is generated with gogo/protobuf.
func (s *LogsBytesSizer) DeltaSize(newItemSize int) int {
	return 1 + newItemSize + math_bits.Len64(uint64(newItemSize|1)+6)/7 //nolint:gosec // disable G115
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
