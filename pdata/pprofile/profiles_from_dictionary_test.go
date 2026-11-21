// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addTenantProfile appends a resource profile to src whose profile references
// a chain of dictionary entries (attribute, stack, location, mapping, function
// and their strings) that belong to name only.
func addTenantProfile(src Profiles, name string) {
	d := src.Dictionary()
	str := func(v string) int32 {
		d.StringTable().Append(v)
		return int32(d.StringTable().Len() - 1)
	}
	attrIdx := int32(d.AttributeTable().Len())
	attr := d.AttributeTable().AppendEmpty()
	attr.SetKeyStrindex(str(name + "-key"))
	attr.Value().SetStr(name + "-value")
	fnIdx := int32(d.FunctionTable().Len())
	d.FunctionTable().AppendEmpty().SetNameStrindex(str(name + "-function"))
	maIdx := int32(d.MappingTable().Len())
	d.MappingTable().AppendEmpty().SetFilenameStrindex(str(name + "-file"))
	locIdx := int32(d.LocationTable().Len())
	loc := d.LocationTable().AppendEmpty()
	loc.SetMappingIndex(maIdx)
	loc.Lines().AppendEmpty().SetFunctionIndex(fnIdx)
	loc.AttributeIndices().Append(attrIdx)
	stIdx := int32(d.StackTable().Len())
	d.StackTable().AppendEmpty().LocationIndices().Append(locIdx)

	prof := src.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	prof.AttributeIndices().Append(attrIdx)
	prof.Samples().AppendEmpty().SetStackIndex(stIdx)
}

func TestNewProfilesFromDictionary(t *testing.T) {
	src := newConformantProfiles()
	addTenantProfile(src, "a")
	addTenantProfile(src, "b")
	before := NewProfilesDictionary()
	src.Dictionary().CopyTo(before)

	for i, name := range []string{"a", "b"} {
		rps := NewResourceProfilesSlice()
		src.ResourceProfiles().At(i).CopyTo(rps.AppendEmpty())

		got, err := NewProfilesFromDictionary(src.Dictionary(), rps)
		require.NoError(t, err)
		assert.Equal(t, 0, rps.Len(), "resource profiles must be moved")

		// The index 0 empty entries plus exactly this tenant's entries.
		d := got.Dictionary()
		assert.ElementsMatch(t, []string{"", name + "-key", name + "-file", name + "-function"}, d.StringTable().AsRaw())
		for _, n := range []int{d.AttributeTable().Len(), d.StackTable().Len(), d.LocationTable().Len(), d.FunctionTable().Len(), d.MappingTable().Len()} {
			assert.Equal(t, 2, n)
		}
		assert.Equal(t, 1, d.LinkTable().Len())

		// References resolve to this tenant's values.
		prof := got.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
		attr := d.AttributeTable().At(int(prof.AttributeIndices().At(0)))
		assert.Equal(t, name+"-key", d.StringTable().At(int(attr.KeyStrindex())))
		assert.Equal(t, name+"-value", attr.Value().Str())
		loc := d.LocationTable().At(int(d.StackTable().At(int(prof.Samples().At(0).StackIndex())).LocationIndices().At(0)))
		assert.Equal(t, name+"-file", d.StringTable().At(int(d.MappingTable().At(int(loc.MappingIndex())).FilenameStrindex())))
		assert.Equal(t, name+"-function", d.StringTable().At(int(d.FunctionTable().At(int(loc.Lines().At(0).FunctionIndex())).NameStrindex())))
		assert.Equal(t, prof.AttributeIndices().At(0), loc.AttributeIndices().At(0), "shared attribute must be deduplicated")
	}

	assert.Equal(t, before.orig, src.Dictionary().orig, "source dictionary must not be modified")
}

func TestNewProfilesFromDictionary_Empty(t *testing.T) {
	src := newConformantProfiles()
	addTenantProfile(src, "a")

	got, err := NewProfilesFromDictionary(src.Dictionary(), NewResourceProfilesSlice())
	require.NoError(t, err)
	assert.Equal(t, 0, got.ResourceProfiles().Len())
	assert.Equal(t, []string{""}, got.Dictionary().StringTable().AsRaw())
	assert.Equal(t, 1, got.Dictionary().AttributeTable().Len())
	assert.Equal(t, 1, got.Dictionary().StackTable().Len())
}

func TestNewProfilesFromDictionary_InvalidIndex(t *testing.T) {
	src := newConformantProfiles()
	rps := NewResourceProfilesSlice()
	rps.AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty().SetStackIndex(99)

	_, err := NewProfilesFromDictionary(src.Dictionary(), rps)
	require.ErrorContains(t, err, "invalid stack index 99")
}
