// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"bytes"
	"fmt"

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
		case "profiles":
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

func (ms ProfileContainer) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "startTimeUnixNano", "start_time_unix_nano":
			ms.orig.StartTimeUnixNano = json.ReadUint64(iter)
		case "endTimeUnixNano", "end_time_unix_nano":
			ms.orig.EndTimeUnixNano = json.ReadUint64(iter)
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.orig.Attributes = append(ms.orig.Attributes, json.ReadAttribute(iter))
				return true
			})
		case "droppedAttributesCount", "dropped_attributes_count":
			ms.orig.DroppedAttributesCount = json.ReadUint32(iter)
		case "profileId", "profile_id":
			if err := ms.orig.ProfileId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readProfile.profileId", fmt.Sprintf("parse profile_id:%v", err))
			}
		default:
			iter.Skip()
		}
		return true
	})
}
