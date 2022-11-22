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

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalLogs(ld Logs) ([]byte, error) {
	pb := internal.LogsToProto(internal.Logs(ld))
	return pb.Marshal()
}

func (e *ProtoMarshaler) MarshalResourceLogs(rl ResourceLogs) ([]byte, error) {
	pb := internal.ResourceLogsToProto(internal.ResourceLogs(rl))
	return pb.Marshal()
}

func (e *ProtoMarshaler) MarshalLogRecord(lr LogRecord) ([]byte, error) {
	pb := internal.LogRecordToProto(internal.LogRecord(lr))
	return pb.Marshal()
}

func (e *ProtoMarshaler) LogsSize(ld Logs) int {
	pb := internal.LogsToProto(internal.Logs(ld))
	return pb.Size()
}

func (e *ProtoMarshaler) ResourceLogsSize(rl ResourceLogs) int {
	pb := internal.ResourceLogsToProto(internal.ResourceLogs(rl))

	return pb.Size()
}

func (e *ProtoMarshaler) LogRecordSize(lr LogRecord) int {
	pb := internal.LogRecordToProto(internal.LogRecord(lr))

	return pb.Size()
}

var _ Unmarshaler = (*ProtoUnmarshaler)(nil)

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalLogs(buf []byte) (Logs, error) {
	pb := otlplogs.LogsData{}
	err := pb.Unmarshal(buf)
	return Logs(internal.LogsFromProto(pb)), err
}

func (d *ProtoUnmarshaler) UnmarshalResourceLogs(buf []byte) (ResourceLogs, error) {
	pb := otlplogs.ResourceLogs{}
	err := pb.Unmarshal(buf)
	return ResourceLogs(internal.ResourceLogsFromProto(pb)), err
}

func (d *ProtoUnmarshaler) UnmarshalLogRecord(buf []byte) (LogRecord, error) {
	pb := otlplogs.LogRecord{}
	err := pb.Unmarshal(buf)
	return LogRecord(internal.LogRecordFromProto(pb)), err
}
