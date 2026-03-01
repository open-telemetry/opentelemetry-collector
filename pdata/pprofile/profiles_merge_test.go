// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilesMergeTo(t *testing.T) {
	src := NewProfiles()
	dest := NewProfiles()

	src.ResourceProfiles().AppendEmpty()
	src.ResourceProfiles().AppendEmpty()
	dest.ResourceProfiles().AppendEmpty()

	require.NoError(t, src.MergeTo(dest))

	assert.Equal(t, 3, dest.ResourceProfiles().Len())
	assert.Equal(t, 0, src.ResourceProfiles().Len())
	assert.True(t, src.IsReadOnly())
}

func TestProfilesMergeToSelf(t *testing.T) {
	profiles := NewProfiles()
	profiles.Dictionary().StringTable().Append("", "test")
	profiles.ResourceProfiles().AppendEmpty()

	require.NoError(t, profiles.MergeTo(profiles))

	assert.Equal(t, 2, profiles.Dictionary().StringTable().Len())
	assert.Equal(t, 1, profiles.ResourceProfiles().Len())
}

func TestProfilesMergeToError(t *testing.T) {
	src := NewProfiles()
	dest := NewProfiles()

	stackTable := src.Dictionary().StackTable()
	stackTable.AppendEmpty()
	stack := stackTable.AppendEmpty()
	stack.LocationIndices().Append(1)

	locationTable := src.Dictionary().LocationTable()
	locationTable.AppendEmpty()
	locationTable.AppendEmpty().SetMappingIndex(1)

	sample := src.ResourceProfiles().AppendEmpty().
		ScopeProfiles().AppendEmpty().
		Profiles().AppendEmpty().
		Samples().AppendEmpty()
	sample.SetStackIndex(1)

	err := src.MergeTo(dest)
	require.Error(t, err)

	assert.Equal(t, 0, dest.ResourceProfiles().Len())
}

func TestProfilesMergeToCanReuseSource(t *testing.T) {
	// This test verifies that switchDictionary does not mutate the source dictionary tables.
	// Without proper copying, dictionary table entries are modified in-place, corrupting the source.
	src := NewProfiles()

	// Set up string table
	src.Dictionary().StringTable().Append("", "attrKey", "attrValue", "funcName", "fileName")

	// Set up function table - index 0 is reserved, so add a dummy first
	src.Dictionary().FunctionTable().AppendEmpty()       // Index 0 (reserved as "not set")
	fn := src.Dictionary().FunctionTable().AppendEmpty() // Index 1
	fn.SetNameStrindex(3)                                // "funcName" (index 3 in string table)
	fn.SetFilenameStrindex(4)                            // "fileName" (index 4 in string table)

	// Set up location table - index 0 is reserved, so add a dummy first
	src.Dictionary().LocationTable().AppendEmpty()        // Index 0 (reserved as "not set")
	loc := src.Dictionary().LocationTable().AppendEmpty() // Index 1
	line := loc.Lines().AppendEmpty()
	line.SetFunctionIndex(1) // Index 1 in function table

	// Set up stack table - index 0 is reserved, so add a dummy first
	src.Dictionary().StackTable().AppendEmpty()          // Index 0 (reserved as "not set")
	stack := src.Dictionary().StackTable().AppendEmpty() // Index 1
	stack.LocationIndices().Append(1)                    // Index 1 in location table

	// Set up attribute table - index 0 is reserved, so add a dummy first
	src.Dictionary().AttributeTable().AppendEmpty()         // Index 0 (reserved as "not set")
	attr := src.Dictionary().AttributeTable().AppendEmpty() // Index 1
	attr.SetKeyStrindex(1)                                  // "attrKey" (index 1 in string table)

	// Create a profile with a sample that uses all the dictionary entries
	profile := src.ResourceProfiles().AppendEmpty().
		ScopeProfiles().AppendEmpty().
		Profiles().AppendEmpty()
	profile.AttributeIndices().Append(1) // Index 1 in attribute table

	sample := profile.Samples().AppendEmpty()
	sample.AttributeIndices().Append(1) // Index 1 in attribute table
	sample.SetStackIndex(1)             // Index 1 in stack table

	// Save the original dictionary state before merge
	// These should NOT change after switchDictionary is called
	origAttrKeyIndex := src.Dictionary().AttributeTable().At(1).KeyStrindex()
	origFuncNameIndex := src.Dictionary().FunctionTable().At(1).NameStrindex()
	origFuncFileIndex := src.Dictionary().FunctionTable().At(1).FilenameStrindex()
	origLineFuncIndex := src.Dictionary().LocationTable().At(1).Lines().At(0).FunctionIndex()

	dest := NewProfiles()
	dest.Dictionary().StringTable().Append("", "different", "strings", "here")

	require.NoError(t, src.MergeTo(dest))

	afterAttrKeyIndex := src.Dictionary().AttributeTable().At(1).KeyStrindex()

	// Verify the source dictionary tables were NOT mutated
	// (they should still reference the source's StringTable/FunctionTable indices)
	assert.Equal(t, origAttrKeyIndex, afterAttrKeyIndex,
		"AttributeTable entry was mutated - KeyStrindex changed")
	assert.Equal(t, origFuncNameIndex, src.Dictionary().FunctionTable().At(1).NameStrindex(),
		"FunctionTable entry was mutated - NameStrindex changed")
	assert.Equal(t, origFuncFileIndex, src.Dictionary().FunctionTable().At(1).FilenameStrindex(),
		"FunctionTable entry was mutated - FilenameStrindex changed")
	assert.Equal(t, origLineFuncIndex, src.Dictionary().LocationTable().At(1).Lines().At(0).FunctionIndex(),
		"LocationTable Line entry was mutated - FunctionIndex changed")

	// Verify destination received the data with properly remapped indices
	assert.Equal(t, 1, dest.ResourceProfiles().Len())
	assert.Greater(t, dest.Dictionary().StringTable().Len(), 3)
	assert.Positive(t, dest.Dictionary().AttributeTable().Len())

	// Verify the function table entries in destination correctly resolve to the expected strings
	// The function table should have been populated during switchDictionary when processing locations/lines
	require.Positive(t, dest.Dictionary().FunctionTable().Len(),
		"Destination FunctionTable should have entries after merge")
	// Function might be at index 0 or 1 depending on whether dest includes dummy entries
	// Find the function with our expected values
	found := false
	for i := 0; i < dest.Dictionary().FunctionTable().Len(); i++ {
		destFn := dest.Dictionary().FunctionTable().At(i)
		if destFn.NameStrindex() > 0 && destFn.FilenameStrindex() > 0 {
			destFuncNameIdx := destFn.NameStrindex()
			destFuncFileIdx := destFn.FilenameStrindex()
			if dest.Dictionary().StringTable().At(int(destFuncNameIdx)) == "funcName" &&
				dest.Dictionary().StringTable().At(int(destFuncFileIdx)) == "fileName" {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Destination FunctionTable should contain function with 'funcName' and 'fileName'")
}
