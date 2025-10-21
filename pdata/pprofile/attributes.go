// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

type attributable interface {
	AttributeIndices() pcommon.Int32Slice
}

// FromAttributeIndices builds a [pcommon.Map] containing the attributes of a
// record.
// The record can be any struct that implements an `AttributeIndices` method.
// Updates made to the return map will not be applied back to the record.
func FromAttributeIndices(table KeyValueAndUnitSlice, record attributable, dic ProfilesDictionary) pcommon.Map {
	m := pcommon.NewMap()
	m.EnsureCapacity(record.AttributeIndices().Len())

	for i := 0; i < record.AttributeIndices().Len(); i++ {
		kv := table.At(int(record.AttributeIndices().At(i)))
		key := dic.StringTable().At(int(kv.KeyStrindex()))
		kv.Value().CopyTo(m.PutEmpty(key))
	}

	return m
}

var errTooManyAttributeTableEntries = errors.New("too many entries in AttributeTable")

// PutAttribute updates an AttributeTable and a record's AttributeIndices to
// add or update an attribute.
// The assumption is that attributes are a map as for other signals (metrics, logs, etc.), thus
// the same key must not appear twice in a list of attributes / attribute indices.
// The record can be any struct that implements an `AttributeIndices` method.
//
// Deprecated: [v0.138.0] use SetAttribute instead.
func PutAttribute(table KeyValueAndUnitSlice, record attributable, dic ProfilesDictionary, key string, value pcommon.Value) error {
	for i := range record.AttributeIndices().Len() {
		idx := int(record.AttributeIndices().At(i))
		if idx < 0 || idx >= table.Len() {
			return fmt.Errorf("index value %d out of range in AttributeIndices[%d]", idx, i)
		}
		attr := table.At(idx)

		if dic.StringTable().At(int(attr.KeyStrindex())) == key {
			if attr.Value().Equal(value) {
				// Attribute already exists, nothing to do.
				return nil
			}

			// If the attribute table already contains the key/value pair, just update the index.
			for j := range table.Len() {
				a := table.At(j)
				if dic.StringTable().At(int(a.KeyStrindex())) == key && a.Value().Equal(value) {
					if j > math.MaxInt32 {
						return errTooManyAttributeTableEntries
					}
					record.AttributeIndices().SetAt(i, int32(j)) //nolint:gosec // overflow checked
					return nil
				}
			}

			if table.Len() >= math.MaxInt32 {
				return errTooManyAttributeTableEntries
			}

			// Find the key in the StringTable, or add it
			keyID, err := SetString(dic.StringTable(), key)
			if err != nil {
				return err
			}

			// Add the key/value pair as a new attribute to the table...
			entry := table.AppendEmpty()
			entry.SetKeyStrindex(keyID)
			value.CopyTo(entry.Value())

			// ...and update the existing index.
			record.AttributeIndices().SetAt(i, int32(table.Len()-1)) //nolint:gosec // overflow checked
			return nil
		}
	}

	if record.AttributeIndices().Len() >= math.MaxInt32 {
		return errors.New("too many entries in AttributeIndices")
	}

	for j := range table.Len() {
		a := table.At(j)
		if dic.StringTable().At(int(a.KeyStrindex())) == key && a.Value().Equal(value) {
			if j > math.MaxInt32 {
				return errTooManyAttributeTableEntries
			}
			// Add the index of the existing attribute to the indices.
			record.AttributeIndices().Append(int32(j)) //nolint:gosec // overflow checked
			return nil
		}
	}

	if table.Len() >= math.MaxInt32 {
		return errTooManyAttributeTableEntries
	}

	// Find the key in the StringTable, or add it
	keyID, err := SetString(dic.StringTable(), key)
	if err != nil {
		return err
	}

	// Add the key/value pair as a new attribute to the table...
	entry := table.AppendEmpty()
	entry.SetKeyStrindex(keyID)
	value.CopyTo(entry.Value())

	// ...and add a new index to the indices.
	record.AttributeIndices().Append(int32(table.Len() - 1)) //nolint:gosec // overflow checked
	return nil
}

// SetAttribute updates an AttributeTable, adding or providing a value and
// returns its index.
func SetAttribute(table KeyValueAndUnitSlice, attr KeyValueAndUnit) (int32, error) {
	for j, a := range table.All() {
		if a.Equal(attr) {
			if j > math.MaxInt32 {
				return 0, errTooManyAttributeTableEntries
			}
			return int32(j), nil //nolint:gosec // G115 overflow checked
		}
	}

	if table.Len() >= math.MaxInt32 {
		return 0, errTooManyAttributeTableEntries
	}

	attr.CopyTo(table.AppendEmpty())
	return int32(table.Len() - 1), nil //nolint:gosec // G115 overflow checked
}
