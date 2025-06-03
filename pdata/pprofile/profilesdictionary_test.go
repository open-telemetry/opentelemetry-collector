// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfilesDictionary_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expected := generateTestProfilesDictionary()
	dest := NewProfilesDictionary()
	src := generateTestProfilesDictionary()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestProfilesDictionary(), dest)
	assert.Equal(t, 0, src.MappingTable().Len())
	assert.Equal(t, 0, src.LocationTable().Len())
	assert.Equal(t, 0, src.FunctionTable().Len())
	assert.Equal(t, 0, src.LinkTable().Len())
	assert.Equal(t, 0, src.StringTable().Len())
	assert.Equal(t, 0, src.AttributeTable().Len())
	assert.Equal(t, 0, src.AttributeUnits().Len())

	assert.Equal(t, expected.MappingTable().Len(), dest.MappingTable().Len())
	assert.Equal(t, expected.LocationTable().Len(), dest.LocationTable().Len())
	assert.Equal(t, expected.FunctionTable().Len(), dest.FunctionTable().Len())
	assert.Equal(t, expected.LinkTable().Len(), dest.LinkTable().Len())
	assert.Equal(t, expected.StringTable().Len(), dest.StringTable().Len())
	assert.Equal(t, expected.AttributeTable().Len(), dest.AttributeTable().Len())
	assert.Equal(t, expected.AttributeUnits().Len(), dest.AttributeUnits().Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestProfilesDictionary(), dest)
	assert.Equal(t, 0, src.MappingTable().Len())
	assert.Equal(t, 0, src.LocationTable().Len())
	assert.Equal(t, 0, src.FunctionTable().Len())
	assert.Equal(t, 0, src.LinkTable().Len())
	assert.Equal(t, 0, src.StringTable().Len())
	assert.Equal(t, 0, src.AttributeTable().Len())
	assert.Equal(t, 0, src.AttributeUnits().Len())

	assert.Equal(t, expected.MappingTable().Len(), dest.MappingTable().Len())
	assert.Equal(t, expected.LocationTable().Len(), dest.LocationTable().Len())
	assert.Equal(t, expected.FunctionTable().Len(), dest.FunctionTable().Len())
	assert.Equal(t, expected.LinkTable().Len(), dest.LinkTable().Len())
	assert.Equal(t, expected.StringTable().Len(), dest.StringTable().Len())
	assert.Equal(t, expected.AttributeTable().Len(), dest.AttributeTable().Len())
	assert.Equal(t, expected.AttributeUnits().Len(), dest.AttributeUnits().Len())

	// Test MoveAndAppendTo not empty slice
	generateTestProfilesDictionary().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expected.MappingTable().Len(), dest.MappingTable().Len())
	assert.Equal(t, 2*expected.LocationTable().Len(), dest.LocationTable().Len())
	assert.Equal(t, 2*expected.FunctionTable().Len(), dest.FunctionTable().Len())
	assert.Equal(t, 2*expected.LinkTable().Len(), dest.LinkTable().Len())
	assert.Equal(t, 2*expected.StringTable().Len(), dest.StringTable().Len())
	assert.Equal(t, 2*expected.AttributeTable().Len(), dest.AttributeTable().Len())
	assert.Equal(t, 2*expected.AttributeUnits().Len(), dest.AttributeUnits().Len())

	expectEqualDictionaries(t, expected, dest)

	dest.MoveAndAppendTo(dest)
	assert.Equal(t, 2*expected.MappingTable().Len(), dest.MappingTable().Len())
	assert.Equal(t, 2*expected.LocationTable().Len(), dest.LocationTable().Len())
	assert.Equal(t, 2*expected.FunctionTable().Len(), dest.FunctionTable().Len())
	assert.Equal(t, 2*expected.LinkTable().Len(), dest.LinkTable().Len())
	assert.Equal(t, 2*expected.StringTable().Len(), dest.StringTable().Len())
	assert.Equal(t, 2*expected.AttributeTable().Len(), dest.AttributeTable().Len())
	assert.Equal(t, 2*expected.AttributeUnits().Len(), dest.AttributeUnits().Len())

	expectEqualDictionaries(t, expected, dest)
}

func expectEqualDictionaries(t *testing.T, expected, dest ProfilesDictionary) {
	for i := 0; i < expected.MappingTable().Len(); i++ {
		assert.Equal(t, expected.MappingTable().At(i), dest.MappingTable().At(i))
		assert.Equal(t, expected.MappingTable().At(i), dest.MappingTable().At(i+expected.MappingTable().Len()))
	}

	for i := 0; i < expected.LocationTable().Len(); i++ {
		assert.Equal(t, expected.LocationTable().At(i), dest.LocationTable().At(i))
		assert.Equal(t, expected.LocationTable().At(i), dest.LocationTable().At(i+expected.LocationTable().Len()))
	}

	for i := 0; i < expected.FunctionTable().Len(); i++ {
		assert.Equal(t, expected.FunctionTable().At(i), dest.FunctionTable().At(i))
		assert.Equal(t, expected.FunctionTable().At(i), dest.FunctionTable().At(i+expected.FunctionTable().Len()))
	}

	for i := 0; i < expected.LinkTable().Len(); i++ {
		assert.Equal(t, expected.LinkTable().At(i), dest.LinkTable().At(i))
		assert.Equal(t, expected.LinkTable().At(i), dest.LinkTable().At(i+expected.LinkTable().Len()))
	}

	for i := 0; i < expected.StringTable().Len(); i++ {
		assert.Equal(t, expected.StringTable().At(i), dest.StringTable().At(i))
		assert.Equal(t, expected.StringTable().At(i), dest.StringTable().At(i+expected.StringTable().Len()))
	}

	for i := 0; i < expected.AttributeTable().Len(); i++ {
		assert.Equal(t, expected.AttributeTable().At(i), dest.AttributeTable().At(i))
		assert.Equal(t, expected.AttributeTable().At(i), dest.AttributeTable().At(i+expected.AttributeTable().Len()))
	}

	for i := 0; i < expected.AttributeUnits().Len(); i++ {
		assert.Equal(t, expected.AttributeUnits().At(i), dest.AttributeUnits().At(i))
		assert.Equal(t, expected.AttributeUnits().At(i), dest.AttributeUnits().At(i+expected.AttributeUnits().Len()))
	}
}
