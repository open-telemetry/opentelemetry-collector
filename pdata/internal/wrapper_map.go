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
	*pMap
}

type pMap struct {
	orig   *[]otlpcommon.KeyValue
	state  *State
	parent Parent[*[]otlpcommon.KeyValue]
}

func (ms Map) GetOrig() *[]otlpcommon.KeyValue {
	if *ms.state == StateDirty {
		ms.orig, ms.state = ms.parent.RefreshOrigState()
	}
	return ms.orig
}

func (ms Map) EnsureMutability() {
	if *ms.state == StateShared {
		ms.parent.EnsureMutability()
	}
}

func (ms Map) GetState() *State {
	return ms.state
}

func (ms Map) GetValueParent(key string) MapValueParent {
	return MapValueParent{Map: ms, key: key}
}

type MapValueParent struct {
	Map
	key string
}

func (ps MapValueParent) RefreshOrigState() (*otlpcommon.AnyValue, *State) {
	for i := range *ps.GetOrig() {
		akv := &(*ps.GetOrig())[i]
		if akv.Key == ps.key {
			return &akv.Value, ps.state
		}
	}
	return &otlpcommon.AnyValue{}, ps.state
}

type ValueBytes struct {
	Value
}

func (ms ValueBytes) RefreshOrigState() (*[]byte, *State) {
	val := ms.Value.GetOrig().GetBytesValue()
	return &val, ms.Value.GetState()
}

func (ms Map) Get(key string) (Value, bool) {
	for i := range *ms.GetOrig() {
		akv := &(*ms.GetOrig())[i]
		if akv.Key == key {
			return NewValue(&akv.Value, ms.GetValueParent(key)), true
		}
	}
	return NewValue(nil, nil), false
}

func NewMap(orig *[]otlpcommon.KeyValue, parent Parent[*[]otlpcommon.KeyValue]) Map {
	if parent == nil {
		state := StateExclusive
		return Map{&pMap{orig: orig, state: &state}}
	}
	return Map{&pMap{orig: orig, state: parent.GetState(), parent: parent}}
}

func GenerateTestMap() Map {
	var orig []otlpcommon.KeyValue
	ms := NewMap(&orig, nil)
	FillTestMap(ms)
	return ms
}

func FillTestMap(dest Map) {
	*dest.orig = nil
	*dest.orig = append(*dest.orig, otlpcommon.KeyValue{Key: "k", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}}})
}
