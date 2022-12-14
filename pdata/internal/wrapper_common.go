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
	orig *otlpcommon.AnyValue
}

type MutableValue struct {
	orig *otlpcommon.AnyValue
}

func GetOrigValue(ms Value) *otlpcommon.AnyValue {
	return ms.orig
}

func GetMutableOrigValue(ms MutableValue) *otlpcommon.AnyValue {
	return ms.orig
}

func NewValue(orig *otlpcommon.AnyValue) Value {
	return Value{orig: orig}
}

func NewMutableValue(orig *otlpcommon.AnyValue) MutableValue {
	return MutableValue{orig: orig}
}

func FillTestValue(dest MutableValue) {
	dest.orig.Value = &otlpcommon.AnyValue_StringValue{StringValue: "v"}
}

func GenerateTestValue() MutableValue {
	var orig otlpcommon.AnyValue
	ms := NewMutableValue(&orig)
	FillTestValue(ms)
	return ms
}

type Map struct {
	orig *[]otlpcommon.KeyValue
}

type MutableMap struct {
	mo *[]otlpcommon.KeyValue
}

func GetOrigMap(ms Map) *[]otlpcommon.KeyValue {
	return ms.orig
}

func GetMutableOrigMap(ms MutableMap) *[]otlpcommon.KeyValue {
	return ms.mo
}

func NewMap(orig *[]otlpcommon.KeyValue) Map {
	return Map{orig: orig}
}

func NewMutableMap(orig *[]otlpcommon.KeyValue) MutableMap {
	return MutableMap{mo: orig}
}

func GenerateTestMap() MutableMap {
	var orig []otlpcommon.KeyValue
	ms := NewMutableMap(&orig)
	FillTestMap(ms)
	return ms
}

func FillTestMap(dest MutableMap) {
	*dest.mo = nil
	*dest.mo = append(*dest.mo, otlpcommon.KeyValue{Key: "k", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}}})
}

type TraceState struct {
	orig *string
}

func GetOrigTraceState(ms TraceState) *string {
	return ms.orig
}

func NewTraceState(orig *string) TraceState {
	return TraceState{orig: orig}
}

func GenerateTestTraceState() TraceState {
	var orig string
	ms := NewTraceState(&orig)
	FillTestTraceState(ms)
	return ms
}

func FillTestTraceState(dest TraceState) {
	*dest.orig = "rojo=00f067aa0ba902b7"
}
