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

type Value struct {
	*pValue
}

type pValue struct {
	orig   *otlpcommon.AnyValue
	state  *State
	parent Parent[*otlpcommon.AnyValue]
}

func (ms Value) EnsureMutability() {
	if *ms.state == StateShared {
		ms.parent.EnsureMutability()
	}
}

func (ms Value) GetState() *State {
	return ms.state
}

func (ms Value) GetOrig() *otlpcommon.AnyValue {
	if *ms.state == StateDirty {
		ms.orig, ms.state = ms.parent.RefreshOrigState()
	}
	return ms.orig
}

type ValueMap struct {
	Value
}

func (ms ValueMap) GetChildOrig() *[]otlpcommon.KeyValue {
	return &ms.Value.GetOrig().GetKvlistValue().Values
}

type ValueSlice struct {
	Value
}

func (ms ValueSlice) GetChildOrig() *[]otlpcommon.AnyValue {
	return &ms.Value.GetOrig().GetArrayValue().Values
}

func NewValue(orig *otlpcommon.AnyValue, parent Parent[*otlpcommon.AnyValue]) Value {
	if parent == nil {
		state := StateExclusive
		return Value{&pValue{orig: orig, state: &state, parent: parent}}
	}
	return Value{&pValue{orig: orig, state: parent.GetState(), parent: parent}}
}

func FillTestValue(dest Value) {
	dest.orig.Value = &otlpcommon.AnyValue_StringValue{StringValue: "v"}
}

func GenerateTestValue() Value {
	var orig otlpcommon.AnyValue
	ms := NewValue(&orig, nil)
	FillTestValue(ms)
	return ms
}
