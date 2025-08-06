// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
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
	var newDest []otlpcommon.KeyValue
	if cap(dest) < len(src) {
		newDest = make([]otlpcommon.KeyValue, len(src))
	} else {
		newDest = dest[:len(src)]
		// Cleanup the rest of the elements so GC can free the memory.
		// This can happen when len(src) < len(dest) < cap(dest).
		for i := len(src); i < len(dest); i++ {
			dest[i] = otlpcommon.KeyValue{}
		}
	}
	for i := range src {
		CopyOrigKeyValue(&newDest[i], &src[i])
	}
	return newDest
}

func GenerateTestMap() Map {
	orig := GenerateOrigTestKeyValueSlice()
	state := StateMutable
	return NewMap(&orig, &state)
}

// MarshalJSONStreamMap marshals all properties from the current struct to the destination stream.
func MarshalJSONStreamMap(ms Map, dest *json.Stream) {
	dest.WriteArrayStart()
	if len(*ms.orig) > 0 {
		writeAttribute(&(*ms.orig)[0], ms.state, dest)
	}
	for i := 1; i < len(*ms.orig); i++ {
		dest.WriteMore()
		writeAttribute(&(*ms.orig)[i], ms.state, dest)
	}
	dest.WriteArrayEnd()
}

func writeAttribute(attr *otlpcommon.KeyValue, state *State, dest *json.Stream) {
	dest.WriteObjectStart()
	if attr.Key != "" {
		dest.WriteObjectField("key")
		dest.WriteString(attr.Key)
	}
	dest.WriteObjectField("value")
	MarshalJSONStreamValue(NewValue(&attr.Value, state), dest)
	dest.WriteObjectEnd()
}

func UnmarshalJSONIterMap(ms Map, iter *json.Iterator) {
	iter.ReadArrayCB(func(iter *json.Iterator) bool {
		*ms.orig = append(*ms.orig, otlpcommon.KeyValue{})
		iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
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
