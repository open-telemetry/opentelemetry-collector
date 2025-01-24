// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type attributable interface {
	AttributeIndices() pcommon.Int32Slice
}

// BuildAttributes builds a [pcommon.Map] containing the attributes of a record.
// The record can by any struct that implements an `AttributeIndices` method.
func BuildAttributes(profile Profile, record attributable) pcommon.Map {
	m := pcommon.NewMap()

	for i := 0; i < record.AttributeIndices().Len(); i++ {
		id := int(record.AttributeIndices().At(i))
		a := profile.AttributeTable().At(id)

		key := a.Key()
		val := a.Value()

		switch val.Type() {
		case pcommon.ValueTypeEmpty:
			m.PutStr(key, "")
		case pcommon.ValueTypeStr:
			m.PutStr(key, val.AsString())
		case pcommon.ValueTypeBool:
			m.PutBool(key, val.Bool())
		case pcommon.ValueTypeDouble:
			m.PutDouble(key, val.Double())
		case pcommon.ValueTypeInt:
			m.PutInt(key, val.Int())
		case pcommon.ValueTypeBytes:
			bs := m.PutEmptyBytes(key)
			val.Bytes().CopyTo(bs)
		case pcommon.ValueTypeMap:
			ma := m.PutEmptyMap(key)
			val.Map().CopyTo(ma)
		case pcommon.ValueTypeSlice:
			sl := m.PutEmptySlice(key)
			val.Slice().CopyTo(sl)
		}
	}

	return m
}
