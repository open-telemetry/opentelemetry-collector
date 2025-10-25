// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalLogs(ld Logs) ([]byte, error) {
	size := internal.SizeProtoExportLogsServiceRequest(ld.getOrig())
	buf := make([]byte, size)
	_ = internal.MarshalProtoExportLogsServiceRequest(ld.getOrig(), buf)
	return buf, nil
}

func (e *ProtoMarshaler) LogsSize(ld Logs) int {
	return internal.SizeProtoExportLogsServiceRequest(ld.getOrig())
}

func (e *ProtoMarshaler) ResourceLogsSize(ld ResourceLogs) int {
	return internal.SizeProtoResourceLogs(ld.orig)
}

func (e *ProtoMarshaler) ScopeLogsSize(ld ScopeLogs) int {
	return internal.SizeProtoScopeLogs(ld.orig)
}

func (e *ProtoMarshaler) LogRecordSize(ld LogRecord) int {
	return internal.SizeProtoLogRecord(ld.orig)
}

var _ Unmarshaler = (*ProtoUnmarshaler)(nil)

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalLogs(buf []byte) (Logs, error) {
	ld := NewLogs()
	err := internal.UnmarshalProtoExportLogsServiceRequest(ld.getOrig(), buf)
	if err != nil {
		return Logs{}, err
	}
	return ld, nil
}
