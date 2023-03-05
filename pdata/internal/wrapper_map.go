// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type Map struct {
	parent Parent[*[]otlpcommon.KeyValue]
}

type stubMapParent struct {
	orig *[]otlpcommon.KeyValue
}

func (mp stubMapParent) EnsureMutability() {}

func (mp stubMapParent) GetChildOrig() *[]otlpcommon.KeyValue {
	return mp.orig
}

var _ Parent[*[]otlpcommon.KeyValue] = (*stubMapParent)(nil)

func (ms Map) GetOrig() *[]otlpcommon.KeyValue {
	return ms.parent.GetChildOrig()
}

func (ms Map) EnsureMutability() {
	ms.parent.EnsureMutability()
}

func (ms Map) GetValueParent(key string) MapValueParent {
	return MapValueParent{Map: ms, key: key}
}

type MapValueParent struct {
	Map
	key string
}

func (ps MapValueParent) GetChildOrig() *otlpcommon.AnyValue {
	for i := range *ps.GetOrig() {
		akv := &(*ps.GetOrig())[i]
		if akv.Key == ps.key {
			return &akv.Value
		}
	}
	return &otlpcommon.AnyValue{}
}

type ValueBytes struct {
	Value
}

func (ms ValueBytes) GetChildOrig() *[]byte {
	return &ms.Value.GetOrig().GetValue().(*otlpcommon.AnyValue_BytesValue).BytesValue
}

func (ms Map) Get(key string) (Value, bool) {
	for i := range *ms.GetOrig() {
		akv := &(*ms.GetOrig())[i]
		if akv.Key == key {
			return NewValueFromParent(ms.GetValueParent(key)), true
		}
	}
	return Value{}, false
}

func NewMapFromOrig(orig *[]otlpcommon.KeyValue) Map {
	return Map{parent: &stubMapParent{orig: orig}}
}

func NewMapFromParent(parent Parent[*[]otlpcommon.KeyValue]) Map {
	return Map{parent: parent}
}

func GenerateTestMap() Map {
	var orig []otlpcommon.KeyValue
	ms := NewMapFromOrig(&orig)
	FillTestMap(ms)
	return ms
}

func FillTestMap(dest Map) {
	*dest.GetOrig() = nil
	*dest.GetOrig() = append(*dest.GetOrig(), otlpcommon.KeyValue{Key: "k", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}}})
}
