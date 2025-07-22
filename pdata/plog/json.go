// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"fmt"
	"slices"

	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pdata.Logs to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalLogs to the OTLP/JSON format.
func (*JSONMarshaler) MarshalLogs(ld Logs) ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	ld.marshalJSONStream(dest)
	return slices.Clone(dest.Buffer()), dest.Error
}

var _ Unmarshaler = (*JSONUnmarshaler)(nil)

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to pdata.Logs.
type JSONUnmarshaler struct{}

// UnmarshalLogs from OTLP/JSON format into pdata.Logs.
func (*JSONUnmarshaler) UnmarshalLogs(buf []byte) (Logs, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	ld := NewLogs()
	ld.unmarshalJsoniter(iter)
	if iter.Error != nil {
		return Logs{}, iter.Error
	}
	otlp.MigrateLogs(ld.getOrig().ResourceLogs)
	return ld, nil
}

func (ms ResourceLogs) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			internal.UnmarshalJSONIterResource(internal.NewResource(&ms.orig.Resource, ms.state), iter)
		case "scope_logs", "scopeLogs":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.ScopeLogs().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ScopeLogs) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			internal.UnmarshalJSONIterInstrumentationScope(internal.NewInstrumentationScope(&ms.orig.Scope, ms.state), iter)
		case "log_records", "logRecords":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.LogRecords().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms LogRecord) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "observed_time_unix_nano", "observedTimeUnixNano":
			ms.orig.ObservedTimeUnixNano = json.ReadUint64(iter)
		case "severity_number", "severityNumber":
			ms.orig.SeverityNumber = otlplogs.SeverityNumber(json.ReadEnumValue(iter, otlplogs.SeverityNumber_value))
		case "severity_text", "severityText":
			ms.orig.SeverityText = iter.ReadString()
		case "event_name", "eventName":
			ms.orig.EventName = iter.ReadString()
		case "body":
			internal.UnmarshalJSONIterValue(internal.NewValue(&ms.orig.Body, ms.state), iter)
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "droppedAttributesCount", "dropped_attributes_count":
			ms.orig.DroppedAttributesCount = json.ReadUint32(iter)
		case "flags":
			ms.orig.Flags = json.ReadUint32(iter)
		case "traceId", "trace_id":
			if err := ms.orig.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readLog.traceId", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := ms.orig.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readLog.spanId", fmt.Sprintf("parse span_id:%v", err))
			}
		default:
			iter.Skip()
		}
		return true
	})
}
