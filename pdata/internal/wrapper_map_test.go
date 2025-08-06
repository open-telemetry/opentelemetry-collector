// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestReadAttributeUnknownField(t *testing.T) {
	jsonStr := `[{"extra":""}]`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)
	var got []otlpcommon.KeyValue
	UnmarshalJSONIterMap(NewMap(&got, nil), iter)
	//  unknown fields should not be an error
	require.NoError(t, iter.Error())
	assert.Equal(t, []otlpcommon.KeyValue{{}}, got)
}

func TestReadAttributeValueUnknownField(t *testing.T) {
	// Key after value, to check that we correctly continue to process.
	jsonStr := `[{"value": {"unknown": {"extra":""}}, "key":"test"}]`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)
	var got []otlpcommon.KeyValue
	UnmarshalJSONIterMap(NewMap(&got, nil), iter)
	//  unknown fields should not be an error
	require.NoError(t, iter.Error())
	assert.Equal(t, []otlpcommon.KeyValue{{Key: "test"}}, got)
}

func TestMarshalAndUnmarshalJSONMap(t *testing.T) {
	stream := json.BorrowStream(nil)
	defer json.ReturnStream(stream)

	state := StateMutable
	src := NewMap(&[]otlpcommon.KeyValue{}, &state)
	fillTestMapAll(src)
	MarshalJSONStreamMap(src, stream)
	require.NoError(t, stream.Error())

	iter := json.BorrowIterator(stream.Buffer())
	defer json.ReturnIterator(iter)
	dest := NewMap(&[]otlpcommon.KeyValue{}, &state)
	UnmarshalJSONIterMap(dest, iter)
	require.NoError(t, iter.Error())

	assert.Equal(t, src, dest)
}

func fillTestMapAll(dest Map) {
	*dest.orig = []otlpcommon.KeyValue{
		{Key: "empty", Value: otlpcommon.AnyValue{Value: nil}},
		{Key: "str", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value"}}},
		{Key: "bool", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BoolValue{BoolValue: true}}},
		{Key: "double", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: 3.14}}},
		{Key: "int", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: 123}}},
		{Key: "bytes", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BytesValue{BytesValue: []byte{1, 2, 3}}}},
		{Key: "array", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{
			ArrayValue: &otlpcommon.ArrayValue{Values: []otlpcommon.AnyValue{{Value: &otlpcommon.AnyValue_IntValue{IntValue: 321}}}},
		}}},
		{Key: "map", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{
			KvlistValue: &otlpcommon.KeyValueList{Values: []otlpcommon.KeyValue{{Key: "key", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value"}}}}},
		}}},
	}
}
