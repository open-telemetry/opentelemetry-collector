// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestReadValueUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)
	value := otlpcommon.AnyValue{}
	UnmarshalJSONOrigAnyValue(&value, iter)
	require.NoError(t, iter.Error())
	assert.Equal(t, otlpcommon.AnyValue{}, value)
}

func TestReadValueInvliadBytesValue(t *testing.T) {
	jsonStr := `{"bytesValue": "--"}`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)

	value := otlpcommon.AnyValue{}
	UnmarshalJSONOrigAnyValue(&value, iter)
	assert.ErrorContains(t, iter.Error(), "base64")
}

func TestReadArrayValueInvalidArrayValue(t *testing.T) {
	jsonStr := `{"arrayValue": {"extra":""}}`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)

	value := otlpcommon.AnyValue{}
	UnmarshalJSONOrigAnyValue(&value, iter)
	require.NoError(t, iter.Error())
	assert.Equal(t, otlpcommon.AnyValue{
		Value: &otlpcommon.AnyValue_ArrayValue{
			ArrayValue: &otlpcommon.ArrayValue{},
		},
	}, value)
}

func TestReadKvlistValueInvalidArrayValue(t *testing.T) {
	jsonStr := `{"kvlistValue": {"extra":""}}`
	iter := json.BorrowIterator([]byte(jsonStr))
	defer json.ReturnIterator(iter)

	value := otlpcommon.AnyValue{}
	UnmarshalJSONOrigAnyValue(&value, iter)
	require.NoError(t, iter.Error())
	assert.Equal(t, otlpcommon.AnyValue{
		Value: &otlpcommon.AnyValue_KvlistValue{
			KvlistValue: &otlpcommon.KeyValueList{},
		},
	}, value)
}

func TestCopyOrigAnyValueAllTypes(t *testing.T) {
	for name, src := range allAnyValues() {
		for _, pooling := range []bool{true, false} {
			t.Run(name+"pooling_"+strconv.FormatBool(pooling), func(t *testing.T) {
				prevPooling := UseProtoPooling.IsEnabled()
				require.NoError(t, featuregate.GlobalRegistry().Set(UseProtoPooling.ID(), pooling))
				defer func() {
					require.NoError(t, featuregate.GlobalRegistry().Set(UseProtoPooling.ID(), prevPooling))
				}()

				dest := NewOrigAnyValue()
				CopyOrigAnyValue(dest, src)
				assert.Equal(t, src, dest)
				DeleteOrigAnyValue(dest, true)
			})
		}
	}
}

func TestMarshalAndUnmarshalJSONOrigAnyValueAllTypes(t *testing.T) {
	for name, src := range allAnyValues() {
		for _, pooling := range []bool{true, false} {
			t.Run(name+"pooling_"+strconv.FormatBool(pooling), func(t *testing.T) {
				prevPooling := UseProtoPooling.IsEnabled()
				require.NoError(t, featuregate.GlobalRegistry().Set(UseProtoPooling.ID(), pooling))
				defer func() {
					require.NoError(t, featuregate.GlobalRegistry().Set(UseProtoPooling.ID(), prevPooling))
				}()

				stream := json.BorrowStream(nil)
				defer json.ReturnStream(stream)
				MarshalJSONOrigAnyValue(src, stream)
				require.NoError(t, stream.Error())

				iter := json.BorrowIterator(stream.Buffer())
				defer json.ReturnIterator(iter)
				dest := NewOrigAnyValue()
				UnmarshalJSONOrigAnyValue(dest, iter)
				require.NoError(t, iter.Error())

				assert.Equal(t, src, dest)
				DeleteOrigAnyValue(dest, true)
			})
		}
	}
}

func TestMarshalAndUnmarshalProtoAnyValueAllTypes(t *testing.T) {
	for name, src := range allAnyValues() {
		for _, pooling := range []bool{true, false} {
			t.Run(name+"pooling_"+strconv.FormatBool(pooling), func(t *testing.T) {
				prevPooling := UseProtoPooling.IsEnabled()
				require.NoError(t, featuregate.GlobalRegistry().Set(UseProtoPooling.ID(), pooling))
				defer func() {
					require.NoError(t, featuregate.GlobalRegistry().Set(UseProtoPooling.ID(), prevPooling))
				}()

				buf := make([]byte, SizeProtoOrigAnyValue(src))
				gotSize := MarshalProtoOrigAnyValue(src, buf)
				assert.Equal(t, len(buf), gotSize)

				dest := NewOrigAnyValue()
				require.NoError(t, UnmarshalProtoOrigAnyValue(dest, buf))

				assert.Equal(t, src, dest)
				DeleteOrigAnyValue(dest, true)
			})
		}
	}
}

func allAnyValues() map[string]*otlpcommon.AnyValue {
	return map[string]*otlpcommon.AnyValue{
		"empty":               {Value: nil},
		"str":                 {Value: &otlpcommon.AnyValue_StringValue{StringValue: "value"}},
		"bool":                {Value: &otlpcommon.AnyValue_BoolValue{BoolValue: true}},
		"double":              {Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: 3.14}},
		"int":                 {Value: &otlpcommon.AnyValue_IntValue{IntValue: 123}},
		"bytes":               {Value: &otlpcommon.AnyValue_BytesValue{BytesValue: []byte{1, 2, 3}}},
		"ArrayValue/default":  {Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}},
		"ArrayValue/test":     {Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: GenTestOrigArrayValue()}},
		"KvlistValue/default": {Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}},
		"KvlistValue/test":    {Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: GenTestOrigKeyValueList()}},
	}
}
