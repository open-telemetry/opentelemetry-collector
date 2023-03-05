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
	parent Parent[*otlpcommon.AnyValue]
}

type stubValueParent struct {
	orig *otlpcommon.AnyValue
}

func (vp stubValueParent) EnsureMutability() {}

func (vp stubValueParent) GetChildOrig() *otlpcommon.AnyValue {
	return vp.orig
}

var _ Parent[*otlpcommon.AnyValue] = (*stubValueParent)(nil)

func (ms Value) EnsureMutability() {
	ms.parent.EnsureMutability()
}

func (ms Value) GetOrig() *otlpcommon.AnyValue {
	return ms.parent.GetChildOrig()
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

func NewValueFromOrig(orig *otlpcommon.AnyValue) Value {
	return Value{parent: &stubValueParent{orig: orig}}
}

func NewValueFromParent(parent Parent[*otlpcommon.AnyValue]) Value {
	return Value{parent: parent}
}

func FillTestValue(dest Value) {
	dest.GetOrig().Value = &otlpcommon.AnyValue_StringValue{StringValue: "v"}
}

func GenerateTestValue() Value {
	var orig otlpcommon.AnyValue
	ms := NewValueFromOrig(&orig)
	FillTestValue(ms)
	return ms
}
