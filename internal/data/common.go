// Copyright 2020 OpenTelemetry Authors
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

package data

// This file contains data structures that are common for all telemetry types,
// such as timestamps, attributes, etc.

import (
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
)

// TimestampUnixNano is a time specified as UNIX Epoch time in nanoseconds since
// 00:00:00 UTC on 1 January 1970.
type TimestampUnixNano uint64

// AttributeValueType specifies the type of value. Numerically is equal to
// otlp.AttributeKeyValue_ValueType.
type AttributeValueType int32

const (
	AttributeValueSTRING AttributeValueType = AttributeValueType(otlpcommon.AttributeKeyValue_STRING)
	AttributeValueINT    AttributeValueType = AttributeValueType(otlpcommon.AttributeKeyValue_INT)
	AttributeValueDOUBLE AttributeValueType = AttributeValueType(otlpcommon.AttributeKeyValue_DOUBLE)
	AttributeValueBOOL   AttributeValueType = AttributeValueType(otlpcommon.AttributeKeyValue_BOOL)
)

// AttributeValue represents a value of an attribute. Typically used in an Attributes map.
// Must use one of NewAttributeValue* functions below to create new instances.
// Important: zero-initialized instance is not valid for use.
//
// Intended to be passed by value since internally it is just a pointer to actual
// value representation. For the same reason passing by value and calling setters
// will modify the original, e.g.:
//
//   function f1(val AttributeValue) { val.SetInt(234) }
//   function f2() {
//   	v := NewAttributeValueString("a string")
//      f1(v)
//      _ := v.GetType() // this will return AttributeValueINT
//   }
type AttributeValue struct {
	orig *otlpcommon.AttributeKeyValue
}

func NewAttributeValueString(v string) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_STRING, StringValue: v}}
}

func NewAttributeValueInt(v int64) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_INT, IntValue: v}}
}

func NewAttributeValueDouble(v float64) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_DOUBLE, DoubleValue: v}}
}

func NewAttributeValueBool(v bool) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_BOOL, BoolValue: v}}
}

// NewAttributeValueSlice creates a slice of attributes values that are correctly initialized.
func NewAttributeValueSlice(len int) []AttributeValue {
	// Allocate 2 slices, one for AttributeValues, another for underlying OTLP structs.
	// TODO: make one allocation for both slices.
	origs := make([]otlpcommon.AttributeKeyValue, len)
	wrappers := make([]AttributeValue, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

// All AttributeValue functions bellow must be called only on instances that are created
// via NewAttributeValue* functions. Calling these functions on zero-initialized
// AttributeValue struct will cause a panic.

func (a AttributeValue) Type() AttributeValueType {
	return AttributeValueType(a.orig.Type)
}

func (a AttributeValue) StringVal() string {
	return a.orig.StringValue
}

func (a AttributeValue) IntVal() int64 {
	return a.orig.IntValue
}

func (a AttributeValue) DoubleVal() float64 {
	return a.orig.DoubleValue
}

func (a AttributeValue) BoolVal() bool {
	return a.orig.BoolValue
}

func (a AttributeValue) SetString(v string) {
	a.orig.Type = otlpcommon.AttributeKeyValue_STRING
	a.orig.StringValue = v
}

func (a AttributeValue) SetInt(v int64) {
	a.orig.Type = otlpcommon.AttributeKeyValue_INT
	a.orig.IntValue = v
}

func (a AttributeValue) SetDouble(v float64) {
	a.orig.Type = otlpcommon.AttributeKeyValue_DOUBLE
	a.orig.DoubleValue = v
}

func (a AttributeValue) SetBool(v bool) {
	a.orig.Type = otlpcommon.AttributeKeyValue_BOOL
	a.orig.BoolValue = v
}

// AttributesMap stores a map of attribute keys to values.
type AttributesMap map[string]AttributeValue

// Attributes stores the map of attributes and a number of dropped attributes.
// Typically used by translator functions to easily pass the pair.
type Attributes struct {
	attrs        AttributesMap
	droppedCount uint32
}

func NewAttributes(m AttributesMap, droppedCount uint32) Attributes {
	return Attributes{m, droppedCount}
}
