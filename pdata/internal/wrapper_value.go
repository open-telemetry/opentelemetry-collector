// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

type ValueWrapper struct {
	orig  *AnyValue
	state *State
}

func GetValueOrig(ms ValueWrapper) *AnyValue {
	return ms.orig
}

func GetValueState(ms ValueWrapper) *State {
	return ms.state
}

func NewValueWrapper(orig *AnyValue, state *State) ValueWrapper {
	return ValueWrapper{orig: orig, state: state}
}

func GenTestValueWrapper() ValueWrapper {
	orig := GenTestAnyValue()
	return NewValueWrapper(orig, NewState())
}

func NewAnyValueStringValue() *AnyValue_StringValue {
	return Alloc[AnyValue_StringValue](nil)
}

func NewAnyValueIntValue() *AnyValue_IntValue {
	return Alloc[AnyValue_IntValue](nil)
}

func NewAnyValueBoolValue() *AnyValue_BoolValue {
	return Alloc[AnyValue_BoolValue](nil)
}

func NewAnyValueDoubleValue() *AnyValue_DoubleValue {
	return Alloc[AnyValue_DoubleValue](nil)
}

func NewAnyValueBytesValue() *AnyValue_BytesValue {
	return Alloc[AnyValue_BytesValue](nil)
}

func NewAnyValueArrayValue() *AnyValue_ArrayValue {
	return Alloc[AnyValue_ArrayValue](nil)
}

func NewAnyValueKvlistValue() *AnyValue_KvlistValue {
	return Alloc[AnyValue_KvlistValue](nil)
}
