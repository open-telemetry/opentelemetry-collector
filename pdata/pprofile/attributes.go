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
// The record can by any struct that implements an `AttributeIndices` method.
// Updates made to the return map will not be applied back to the record.
func FromAttributeIndices(table AttributeTableSlice, record attributable) pcommon.Map {
	m := pcommon.NewMap()
	m.EnsureCapacity(record.AttributeIndices().Len())

	for i := 0; i < record.AttributeIndices().Len(); i++ {
		kv := table.At(int(record.AttributeIndices().At(i)))
		kv.Value().CopyTo(m.PutEmpty(kv.Key()))
	}

	return m
}

// AddAttribute updates an AttributeTable and a record's AttributeIndices to
// add a new attribute.
// The record can by any struct that implements an `AttributeIndices` method.
func AddAttribute(table AttributeTableSlice, record attributable, key string, value pcommon.Value) error {
	for i := range table.Len() {
		a := table.At(i)

		if a.Key() == key && a.Value().Equal(value) {
			if i >= math.MaxInt32 {
				return fmt.Errorf("Attribute %s=%#v has too high an index to be added to AttributeIndices", key, value)
			}

			for j := range record.AttributeIndices().Len() {
				v := record.AttributeIndices().At(j)
				if v == int32(i) { //nolint:gosec // overflow checked
					return nil
				}
			}

			record.AttributeIndices().Append(int32(i)) //nolint:gosec // overflow checked
			return nil
		}
	}

	if table.Len() >= math.MaxInt32 {
		return errors.New("AttributeTable can't take more attributes")
	}
	table.EnsureCapacity(table.Len() + 1)
	entry := table.AppendEmpty()
	entry.SetKey(key)
	value.CopyTo(entry.Value())
	record.AttributeIndices().Append(int32(table.Len()) - 1) //nolint:gosec // overflow checked

	return nil
}

// PutAttribute updates an AttributeTable and a record's AttributeIndices to
// add a new attribute.
// The assumption is that attributes are a map as for other signals (metrics, logs, etc.), thus
// the same key must not appear twice in a list of attributes / attribute indices.
// The record can be any struct that implements an `AttributeIndices` method.
func PutAttribute(table AttributeTableSlice, record attributable, key string, value pcommon.Value) error {
	if record.AttributeIndices().Len() >= math.MaxInt32-1 {
		return errors.New("AttributeTable can't take more attributes")
	}

	for i := range record.AttributeIndices().Len() {
		idx := record.AttributeIndices().At(i)
		attr := table.At(int(idx))
		if attr.Key() == key {
			if attr.Value().Equal(value) {
				// Attribute already exists, nothing to do.
				return nil
			}

			// If the attribute table already contains the key/value pair, just update the index.
			for j := range table.Len() {
				a := table.At(j)
				if a.Key() == key && a.Value().Equal(value) {
					record.AttributeIndices().SetAt(i, int32(j)) //nolint:gosec // overflow checked
					return nil
				}
			}

			// Add the key/value pair as a new attribute to the table...
			entry := table.AppendEmpty()
			entry.SetKey(key)
			value.CopyTo(entry.Value())

			// ...and update the existing index.
			record.AttributeIndices().SetAt(i, int32(table.Len()-1)) //nolint:gosec // overflow checked
			return nil
		}
	}

	// Add the key/value pair as a new attribute to the table...
	entry := table.AppendEmpty()
	entry.SetKey(key)
	value.CopyTo(entry.Value())

	// ...and add a new index to the indices.
	record.AttributeIndices().Append(int32(table.Len() - 1)) //nolint:gosec // overflow checked
	return nil
}
