// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentity // import "go.opentelemetry.io/collector/pdata/pentity"

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

// JSONMarshaler marshals pdata.Entities to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalEntities to the OTLP/JSON format.
func (*JSONMarshaler) MarshalEntities(ld Entities) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.EntitiesToProto(internal.Entities(ld))
	err := json.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

var _ Unmarshaler = (*JSONUnmarshaler)(nil)

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to pdata.Entities.
type JSONUnmarshaler struct{}

// UnmarshalEntities from OTLP/JSON format into pdata.Entities.
func (*JSONUnmarshaler) UnmarshalEntities(buf []byte) (Entities, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	ld := NewEntities()
	ld.unmarshalJsoniter(iter)
	if iter.Error != nil {
		return Entities{}, iter.Error
	}
	return ld, nil
}

func (ms Entities) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource_entities", "resourceEntities":
			iter.ReadArrayCB(func(*jsoniter.Iterator) bool {
				ms.ResourceEntities().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ResourceEntities) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			json.ReadResource(iter, &ms.orig.Resource)
		case "scope_entities", "scopeEntities":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.ScopeEntities().AppendEmpty().unmarshalJsoniter(iter)
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

func (ms ScopeEntities) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			json.ReadScope(iter, &ms.orig.Scope)
		case "log_records", "logRecords":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.EntityEvents().AppendEmpty().unmarshalJsoniter(iter)
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

func (ms EntityEvent) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "entityType", "entity_type":
			ms.orig.EntityType = iter.ReadString()
		case "observed_time_unix_nano", "observedTimeUnixNano":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.orig.Id = append(ms.orig.Id, json.ReadAttribute(iter))
				return true
			})
		// TODO: Add support for other fields.
		default:
			iter.Skip()
		}
		return true
	})
}
