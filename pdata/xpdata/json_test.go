// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xpdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestMarshalAndUnmarshalValue(t *testing.T) {
	for name, src := range genTestEncodingValues() {
		t.Run(name, func(t *testing.T) {
			m := &JSONMarshaler{}
			b, err := m.MarshalValue(src)
			require.NoError(t, err)

			u := &JSONUnmarshaler{}
			dest, err := u.UnmarshalValue(b)
			require.NoError(t, err)

			require.Equal(t, src, dest)
		})
	}
}

func TestUnmarshalValueUnknown(t *testing.T) {
	m := &JSONUnmarshaler{}

	b, err := m.UnmarshalValue([]byte(`{"unknown": "string"}`))
	require.NoError(t, err)
	assert.Equal(t, pcommon.NewValueEmpty(), b)
}

func genTestEncodingValues() map[string]pcommon.Value {
	return map[string]pcommon.Value{
		"empty":               pcommon.NewValueEmpty(),
		"StringValue/default": pcommon.NewValueStr(""),
		"StringValue/test":    pcommon.NewValueStr("test_stringvalue"),
		"BoolValue/default":   pcommon.NewValueBool(false),
		"BoolValue/test":      pcommon.NewValueBool(true),
		"IntValue/default":    pcommon.NewValueInt(0),
		"IntValue/test":       pcommon.NewValueInt(13),
		"DoubleValue/default": pcommon.NewValueDouble(0),
		"DoubleValue/test":    pcommon.NewValueDouble(3.1415926),
		"ArrayValue/default":  pcommon.NewValueSlice(),
		"ArrayValue/test": func() pcommon.Value {
			s := pcommon.NewValueSlice()
			s.Slice().AppendEmpty().SetStr("test1")
			s.Slice().AppendEmpty().SetInt(13)
			s.Slice().AppendEmpty().SetBool(true)
			s.Slice().AppendEmpty().SetDouble(3.1415926)

			s1 := s.Slice().AppendEmpty().SetEmptySlice()
			s1.AppendEmpty().SetStr("nested")

			m := s.Slice().AppendEmpty().SetEmptyMap()
			m.PutStr("key1", "value1")

			return s
		}(),
		"KvlistValue/default": pcommon.NewValueMap(),
		"KvlistValue/test": func() pcommon.Value {
			m := pcommon.NewValueMap()
			m.Map().PutStr("key1", "value1")
			m.Map().PutInt("key2", 13)
			m.Map().PutBool("key3", true)
			m.Map().PutDouble("key4", 3.1415926)
			m.Map().PutEmpty("key5").SetStr("value5")

			s := m.Map().PutEmptySlice("key6")
			s.AppendEmpty().SetStr("nested1")

			m1 := m.Map().PutEmptyMap("key6")
			m1.PutStr("nestedKey1", "nestedValue1")

			return m
		}(),
		"BytesValue/test": func() pcommon.Value {
			v := pcommon.NewValueBytes()
			v.Bytes().FromRaw([]byte{1, 2, 3})
			return v
		}(),
	}
}
