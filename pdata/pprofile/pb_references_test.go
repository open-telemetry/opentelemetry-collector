// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestMarshalUnmarshalWithReferences(t *testing.T) {
	profiles := NewProfiles()

	dict := profiles.Dictionary()
	dict.StringTable().Append("") // index 0, required empty string

	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("service.name", "test-service")
	rp.Resource().Attributes().PutStr("host.name", "test-host")

	sp := rp.ScopeProfiles().AppendEmpty()
	sp.Scope().SetName("test-scope")
	sp.Scope().Attributes().PutStr("scope.attr", "scope-value")

	profile := sp.Profiles().AppendEmpty()
	profile.SetProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})

	// Marshal to proto bytes
	marshaler := ProtoMarshaler{}
	bytes, err := marshaler.MarshalProfiles(profiles)
	require.NoError(t, err)
	require.NotEmpty(t, bytes)

	// Verify that string table was populated (should have more than just the empty string)
	assert.Greater(t, dict.StringTable().Len(), 1, "String table should be populated during marshal")

	// Verify references were created in the resource attributes
	mapOrig := internal.GetMapOrig(internal.MapWrapper(rp.Resource().Attributes()))
	foundRef := false
	for i := 0; i < len(*mapOrig); i++ {
		kv := (*mapOrig)[i]
		if kv.KeyRef != 0 {
			foundRef = true
			break
		}
		// Check if value is a string reference
		if ref, ok := kv.Value.Value.(*internal.AnyValue_StringValueRef); ok && ref.StringValueRef != 0 {
			foundRef = true
			break
		}
	}
	assert.True(t, foundRef, "At least one reference should be created in attributes")

	// Unmarshal from proto bytes
	unmarshaler := ProtoUnmarshaler{}
	profiles2, err := unmarshaler.UnmarshalProfiles(bytes)
	require.NoError(t, err)

	// Verify that the API works correctly - attributes should be accessible as strings
	rp2 := profiles2.ResourceProfiles().At(0)
	serviceNameVal, ok := rp2.Resource().Attributes().Get("service.name")
	assert.True(t, ok, "service.name attribute should exist")
	assert.Equal(t, "test-service", serviceNameVal.Str(), "service.name should be resolved to string")

	hostNameVal, ok := rp2.Resource().Attributes().Get("host.name")
	assert.True(t, ok, "host.name attribute should exist")
	assert.Equal(t, "test-host", hostNameVal.Str(), "host.name should be resolved to string")

	sp2 := rp2.ScopeProfiles().At(0)
	scopeAttrVal, ok := sp2.Scope().Attributes().Get("scope.attr")
	assert.True(t, ok, "scope.attr attribute should exist")
	assert.Equal(t, "scope-value", scopeAttrVal.Str(), "scope.attr should be resolved to string")

	// Verify the string table is preserved
	dict2 := profiles2.Dictionary()
	assert.Greater(t, dict2.StringTable().Len(), 1, "String table should be preserved after unmarshal")
}

func TestMarshalUnmarshalNestedValues(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("") // index 0

	rp := profiles.ResourceProfiles().AppendEmpty()
	attrs := rp.Resource().Attributes()

	kvlist := attrs.PutEmptyMap("nested.map")
	kvlist.PutStr("inner.key1", "inner.value1")
	kvlist.PutStr("inner.key2", "inner.value2")

	arr := attrs.PutEmptySlice("string.array")
	arr.AppendEmpty().SetStr("string1")
	arr.AppendEmpty().SetStr("string2")
	arr.AppendEmpty().SetStr("string3")

	// Marshal and unmarshal
	marshaler := ProtoMarshaler{}
	bytes, err := marshaler.MarshalProfiles(profiles)
	require.NoError(t, err)

	unmarshaler := ProtoUnmarshaler{}
	profiles2, err := unmarshaler.UnmarshalProfiles(bytes)
	require.NoError(t, err)

	// Verify nested map values are accessible
	rp2 := profiles2.ResourceProfiles().At(0)
	kvlist2, ok := rp2.Resource().Attributes().Get("nested.map")
	assert.True(t, ok)
	assert.Equal(t, pcommon.ValueTypeMap, kvlist2.Type())

	innerMap := kvlist2.Map()
	innerVal1, ok := innerMap.Get("inner.key1")
	assert.True(t, ok)
	assert.Equal(t, "inner.value1", innerVal1.Str())

	innerVal2, ok := innerMap.Get("inner.key2")
	assert.True(t, ok)
	assert.Equal(t, "inner.value2", innerVal2.Str())

	// Verify array values are accessible
	arr2, ok := rp2.Resource().Attributes().Get("string.array")
	assert.True(t, ok)
	assert.Equal(t, pcommon.ValueTypeSlice, arr2.Type())

	slice := arr2.Slice()
	assert.Equal(t, 3, slice.Len())
	assert.Equal(t, "string1", slice.At(0).Str())
	assert.Equal(t, "string2", slice.At(1).Str())
	assert.Equal(t, "string3", slice.At(2).Str())
}

func TestRoundTripWithReferences(t *testing.T) {
	original := NewProfiles()
	dict := original.Dictionary()
	dict.StringTable().Append("")

	for i := 0; i < 3; i++ {
		rp := original.ResourceProfiles().AppendEmpty()
		rp.Resource().Attributes().PutStr("resource.id", "resource-"+string(rune('A'+i)))

		for j := 0; j < 2; j++ {
			sp := rp.ScopeProfiles().AppendEmpty()
			sp.Scope().SetName("scope-" + string(rune('X'+j)))
			sp.Scope().Attributes().PutStr("scope.version", "1.0.0")

			profile := sp.Profiles().AppendEmpty()
			profile.SetProfileID([16]byte{byte(i), byte(j)})
		}
	}

	// Marshal
	marshaler := ProtoMarshaler{}
	bytes, err := marshaler.MarshalProfiles(original)
	require.NoError(t, err)

	// Unmarshal
	unmarshaler := ProtoUnmarshaler{}
	restored, err := unmarshaler.UnmarshalProfiles(bytes)
	require.NoError(t, err)

	// Verify structure is preserved
	assert.Equal(t, 3, restored.ResourceProfiles().Len())

	for i := 0; i < 3; i++ {
		rp := restored.ResourceProfiles().At(i)
		resourceID, ok := rp.Resource().Attributes().Get("resource.id")
		assert.True(t, ok)
		assert.Equal(t, "resource-"+string(rune('A'+i)), resourceID.Str())

		assert.Equal(t, 2, rp.ScopeProfiles().Len())
		for j := 0; j < 2; j++ {
			sp := rp.ScopeProfiles().At(j)
			assert.Equal(t, "scope-"+string(rune('X'+j)), sp.Scope().Name())

			scopeVersion, ok := sp.Scope().Attributes().Get("scope.version")
			assert.True(t, ok)
			assert.Equal(t, "1.0.0", scopeVersion.Str())

			assert.Equal(t, 1, sp.Profiles().Len())
		}
	}

	// Verify the string table deduplication worked
	// We should have fewer strings than if everything was duplicated
	dictRestored := restored.Dictionary()
	// At minimum we have: "", "resource.id", "resource-A", "resource-B", "resource-C",
	// "scope.version", "1.0.0" = 7 entries
	// May have more due to scope names
	assert.LessOrEqual(t, dictRestored.StringTable().Len(), 7,
		"String table should deduplicate strings efficiently")
}
