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
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// NewJSONMarshaler returns a Marshaler. Marshals to OTLP json bytes.
func NewJSONMarshaler() Marshaler {
	return &jsonMarshaler{delegate: jsonpb.Marshaler{}}
}

type jsonMarshaler struct {
	delegate jsonpb.Marshaler
}

func (e *jsonMarshaler) MarshalLogs(ld Logs) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.LogsToProto(internal.Logs(ld))
	err := e.delegate.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

type jsonUnmarshaler struct{}

// NewJSONUnmarshaler returns a model.Unmarshaler. Unmarshals from OTLP json bytes.
func NewJSONUnmarshaler() Unmarshaler {
	return &jsonUnmarshaler{}
}

func (jsonUnmarshaler) UnmarshalLogs(buf []byte) (Logs, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	ld := readLogsData(iter)
	if iter.Error != nil {
		return Logs{}, iter.Error
	}
	otlp.MigrateLogs(ld.ResourceLogs)
	return Logs(internal.LogsFromProto(ld)), nil
}

func readLogsData(iter *jsoniter.Iterator) otlplogs.LogsData {
	ld := otlplogs.LogsData{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource_logs", "resourceLogs":
			iter.ReadArrayCB(func(iterator *jsoniter.Iterator) bool {
				ld.ResourceLogs = append(ld.ResourceLogs, readResourceLogs(iter))
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return ld
}

func readResourceLogs(iter *jsoniter.Iterator) *otlplogs.ResourceLogs {
	rs := &otlplogs.ResourceLogs{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			json.ReadResource(iter, &rs.Resource)
		case "scope_logs", "scopeLogs":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				rs.ScopeLogs = append(rs.ScopeLogs,
					readScopeLogs(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			rs.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
	return rs
}

func readScopeLogs(iter *jsoniter.Iterator) *otlplogs.ScopeLogs {
	ils := &otlplogs.ScopeLogs{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			json.ReadScope(iter, &ils.Scope)
		case "log_records", "logRecords":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ils.LogRecords = append(ils.LogRecords, readLog(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			ils.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
	return ils
}

func readLog(iter *jsoniter.Iterator) *otlplogs.LogRecord {
	lr := &otlplogs.LogRecord{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			lr.TimeUnixNano = json.ReadUint64(iter)
		case "observed_time_unix_nano", "observedTimeUnixNano":
			lr.ObservedTimeUnixNano = json.ReadUint64(iter)
		case "severity_number", "severityNumber":
			lr.SeverityNumber = readSeverityNumber(iter)
		case "severity_text", "severityText":
			lr.SeverityText = iter.ReadString()
		case "body":
			json.ReadValue(iter, &lr.Body)
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				lr.Attributes = append(lr.Attributes, json.ReadAttribute(iter))
				return true
			})
		case "droppedAttributesCount", "dropped_attributes_count":
			lr.DroppedAttributesCount = json.ReadUint32(iter)
		case "flags":
			lr.Flags = json.ReadUint32(iter)
		case "traceId", "trace_id":
			if err := lr.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readLog.traceId", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := lr.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readLog.spanId", fmt.Sprintf("parse span_id:%v", err))
			}
		default:
			iter.Skip()
		}
		return true
	})
	return lr
}

func readSeverityNumber(iter *jsoniter.Iterator) otlplogs.SeverityNumber {
	return otlplogs.SeverityNumber(json.ReadEnumValue(iter, otlplogs.SeverityNumber_value))
}
