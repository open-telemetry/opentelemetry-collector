// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	jsoniter "github.com/json-iterator/go"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type Map struct {
	orig  *[]otlpcommon.KeyValue
	state *State
}

func GetOrigMap(ms Map) *[]otlpcommon.KeyValue {
	return ms.orig
}

func GetMapState(ms Map) *State {
	return ms.state
}

func NewMap(orig *[]otlpcommon.KeyValue, state *State) Map {
	return Map{orig: orig, state: state}
}

func CopyOrigMap(dest, src []otlpcommon.KeyValue) []otlpcommon.KeyValue {
	if cap(dest) < len(src) {
		dest = make([]otlpcommon.KeyValue, len(src))
	}
	dest = dest[:len(src)]
	for i := 0; i < len(src); i++ {
		dest[i].Key = src[i].Key
		CopyOrigValue(&dest[i].Value, &src[i].Value)
	}
	return dest
}

func GenerateTestMap() Map {
	var orig []otlpcommon.KeyValue
	state := StateMutable
	ms := NewMap(&orig, &state)
	FillTestMap(ms)
	return ms
}

func FillTestMap(dest Map) {
	*dest.orig = nil
	*dest.orig = append(*dest.orig, otlpcommon.KeyValue{Key: "k", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}}})
}

func UnmarshalJSONIterMap(ms Map, iter *jsoniter.Iterator) {
	iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
		*ms.orig = append(*ms.orig, otlpcommon.KeyValue{})
		iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
			switch f {
			case "key":
				(*ms.orig)[len(*ms.orig)-1].Key = iter.ReadString()
			case "value":
				UnmarshalJSONIterValue(NewValue(&(*ms.orig)[len(*ms.orig)-1].Value, nil), iter)
			default:
				iter.Skip()
			}
			return true
		})
		return true
	})
}
