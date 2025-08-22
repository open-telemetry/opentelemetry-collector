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

func TestMarshalAndUnmarshalJSONOrigAnyValue(t *testing.T) {
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

func TestMarshalAndUnmarshalProtoAnyValueUnknown(t *testing.T) {
	dest := NewOrigAnyValue()
	// message Test { required int64 field = 1313; } encoding { "field": "1234" }
	require.NoError(t, UnmarshalProtoOrigAnyValue(dest, []byte{0x88, 0x52, 0xD2, 0x09}))
	assert.Equal(t, NewOrigAnyValue(), dest)
}

func TestMarshalAndUnmarshalProtoOrigAnyValueFailing(t *testing.T) {
	for name, buf := range genTestFailingUnmarshalProtoValuesAnyValue() {
		t.Run(name, func(t *testing.T) {
			dest := NewOrigAnyValue()
			require.Error(t, UnmarshalProtoOrigAnyValue(dest, buf))
		})
	}
}

func TestMarshalAndUnmarshalProtoAnyValue(t *testing.T) {
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

func genTestFailingUnmarshalProtoValuesAnyValue() map[string][]byte {
	return map[string][]byte{
		"invalid_field":               {0x02},
		"StringValue/wrong_wire_type": {0xc},
		"StringValue/missing_value":   {0xa},
		"BoolValue/wrong_wire_type":   {0x14},
		"BoolValue/missing_value":     {0x10},
		"IntValue/wrong_wire_type":    {0x1c},
		"IntValue/missing_value":      {0x18},
		"DoubleValue/wrong_wire_type": {0x24},
		"DoubleValue/missing_value":   {0x21},
		"ArrayValue/wrong_wire_type":  {0x2c},
		"ArrayValue/missing_value":    {0x2a},
		"KvlistValue/wrong_wire_type": {0x34},
		"KvlistValue/missing_value":   {0x30},
		"BytesValue/wrong_wire_type":  {0x3c},
		"BytesValue/missing_value":    {0x3a},
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
