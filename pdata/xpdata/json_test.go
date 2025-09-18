package xpdata

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestMarshalAndUnmarshalAnyValue(t *testing.T) {
	for name, src := range genTestEncodingValuesAnyValue() {
		t.Run(name, func(t *testing.T) {
			m := &JSONMarshaler{}
			b, err := m.MarshalAnyValue(src)
			require.NoError(t, err)

			u := &JSONUnmarshaler{}
			dest, err := u.UnmarshalAnyValue(b)
			require.NoError(t, err)

			require.Equal(t, src, dest)
		})
	}
}

func genTestEncodingValuesAnyValue() map[string]*otlpcommon.AnyValue {
	return map[string]*otlpcommon.AnyValue{
		"empty":               internal.NewOrigAnyValue(),
		"StringValue/default": {Value: &otlpcommon.AnyValue_StringValue{StringValue: ""}},
		"StringValue/test":    {Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_stringvalue"}},
		"BoolValue/default":   {Value: &otlpcommon.AnyValue_BoolValue{BoolValue: false}},
		"BoolValue/test":      {Value: &otlpcommon.AnyValue_BoolValue{BoolValue: true}},
		"IntValue/default":    {Value: &otlpcommon.AnyValue_IntValue{IntValue: int64(0)}},
		"IntValue/test":       {Value: &otlpcommon.AnyValue_IntValue{IntValue: int64(13)}},
		"DoubleValue/default": {Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: float64(0)}},
		"DoubleValue/test":    {Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: float64(3.1415926)}},
		"ArrayValue/default":  {Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}},
		"ArrayValue/test":     {Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: internal.GenTestOrigArrayValue()}},
		"KvlistValue/default": {Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}},
		"KvlistValue/test":    {Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: internal.GenTestOrigKeyValueList()}},
		"BytesValue/default":  {Value: &otlpcommon.AnyValue_BytesValue{BytesValue: nil}},
		"BytesValue/test":     {Value: &otlpcommon.AnyValue_BytesValue{BytesValue: []byte{1, 2, 3}}},
	}
}
