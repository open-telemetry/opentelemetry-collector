// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalProfiles(ld Profiles) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.ProfilesToProto(internal.Profiles(ld))
	err := json.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

var _ Unmarshaler = (*JSONUnmarshaler)(nil)

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	ld := NewProfiles()
	ld.unmarshalJsoniter(iter)
	if iter.Error != nil {
		return Profiles{}, iter.Error
	}
	otlp.MigrateProfiles(ld.getOrig().ResourceProfiles)
	return ld, nil
}

func (ms Profiles) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource_profiles", "resourceProfiles":
			iter.ReadArrayCB(func(iterator *jsoniter.Iterator) bool {
				ms.ResourceProfiles().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ResourceProfiles) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			json.ReadResource(iter, &ms.orig.Resource)
		case "scope_profiles", "scopeProfiles":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.ScopeProfiles().AppendEmpty().unmarshalJsoniter(iter)
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

func (ms ScopeProfiles) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			json.ReadScope(iter, &ms.orig.Scope)
		case "profile_records", "profileRecords":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Profiles().AppendEmpty().unmarshalJsoniter(iter)
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

func (ms Profile) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		// case "timeUnixNano", "time_unix_nano":
		// 	ms.orig.TimeUnixNano = json.ReadUint64(iter)
		// case "observed_time_unix_nano", "observedTimeUnixNano":
		// 	ms.orig.ObservedTimeUnixNano = json.ReadUint64(iter)
		// case "severity_text", "severityText":
		// 	ms.orig.SeverityText = iter.ReadString()
		// case "body":
		// 	json.ReadValue(iter, &ms.orig.Body)
		// case "attributes":
		// 	iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
		// 		ms.orig.Attributes = append(ms.orig.Attributes, json.ReadAttribute(iter))
		// 		return true
		// 	})
		// case "droppedAttributesCount", "dropped_attributes_count":
		// 	ms.orig.DroppedAttributesCount = json.ReadUint32(iter)
		// case "flags":
		// 	ms.orig.Flags = json.ReadUint32(iter)
		// case "traceId", "trace_id":
		// 	if err := ms.orig.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
		// 		iter.ReportError("readProfile.traceId", fmt.Sprintf("parse trace_id:%v", err))
		// 	}
		// case "spanId", "span_id":
		// 	if err := ms.orig.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
		// 		iter.ReportError("readProfile.spanId", fmt.Sprintf("parse span_id:%v", err))
		// 	}
		default:
			iter.Skip()
		}
		return true
	})
}
