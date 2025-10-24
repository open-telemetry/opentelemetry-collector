// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type ValueWrapper struct {
	orig  *otlpcommon.AnyValue
	state *State
}

func GetValueOrig(ms ValueWrapper) *otlpcommon.AnyValue {
	return ms.orig
}

func GetValueState(ms ValueWrapper) *State {
	return ms.state
}

func NewValueWrapper(orig *otlpcommon.AnyValue, state *State) ValueWrapper {
	return ValueWrapper{orig: orig, state: state}
}

func GenTestValueWrapper() ValueWrapper {
	orig := GenTestAnyValue()
	return NewValueWrapper(orig, NewState())
}

func NewOrigAnyValueStringValue() *otlpcommon.AnyValue_StringValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_StringValue{}
	}
	return ProtoPoolAnyValue_StringValue.Get().(*otlpcommon.AnyValue_StringValue)
}

func NewOrigAnyValueIntValue() *otlpcommon.AnyValue_IntValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_IntValue{}
	}
	return ProtoPoolAnyValue_IntValue.Get().(*otlpcommon.AnyValue_IntValue)
}

func NewOrigAnyValueBoolValue() *otlpcommon.AnyValue_BoolValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_BoolValue{}
	}
	return ProtoPoolAnyValue_BoolValue.Get().(*otlpcommon.AnyValue_BoolValue)
}

func NewOrigAnyValueDoubleValue() *otlpcommon.AnyValue_DoubleValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_DoubleValue{}
	}
	return ProtoPoolAnyValue_DoubleValue.Get().(*otlpcommon.AnyValue_DoubleValue)
}

func NewOrigAnyValueBytesValue() *otlpcommon.AnyValue_BytesValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_BytesValue{}
	}
	return ProtoPoolAnyValue_BytesValue.Get().(*otlpcommon.AnyValue_BytesValue)
}

func NewOrigAnyValueArrayValue() *otlpcommon.AnyValue_ArrayValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_ArrayValue{}
	}
	return ProtoPoolAnyValue_ArrayValue.Get().(*otlpcommon.AnyValue_ArrayValue)
}

func NewOrigAnyValueKvlistValue() *otlpcommon.AnyValue_KvlistValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_KvlistValue{}
	}
	return ProtoPoolAnyValue_KvlistValue.Get().(*otlpcommon.AnyValue_KvlistValue)
}
