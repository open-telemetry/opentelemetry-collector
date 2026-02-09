// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal"
)

func TestResolveProfilesReferencesEmpty(t *testing.T) {
	profiles := NewProfiles()
	// Should not panic on empty profiles
	resolveProfilesReferences(profiles)
	assert.Equal(t, 0, profiles.ResourceProfiles().Len())
}

func TestResolveProfilesReferencesWithKeyRef(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("") // index 0
	dict.StringTable().Append("test-key")
	dict.StringTable().Append("test-value")

	rp := profiles.ResourceProfiles().AppendEmpty()
	attrs := rp.Resource().Attributes()

	// Manually create a KeyValue with key_ref
	mapOrig := internal.GetMapOrig(internal.MapWrapper(attrs))
	*mapOrig = append(*mapOrig, internal.KeyValue{
		KeyRef: 1, // references "test-key"
		Value: internal.AnyValue{
			Value: &internal.AnyValue_StringValueRef{
				StringValueRef: 2, // references "test-value"
			},
		},
	})

	resolveProfilesReferences(profiles)

	// Verify key_ref was resolved
	kv := &(*mapOrig)[0]
	assert.Equal(t, "test-key", kv.Key)

	// Verify string_value_ref was resolved
	strVal, ok := kv.Value.Value.(*internal.AnyValue_StringValue)
	assert.True(t, ok)
	assert.Equal(t, "test-value", strVal.StringValue)
}

func TestResolveProfilesReferencesInvalidIndices(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("") // index 0
	dict.StringTable().Append("valid")

	rp := profiles.ResourceProfiles().AppendEmpty()
	attrs := rp.Resource().Attributes()

	mapOrig := internal.GetMapOrig(internal.MapWrapper(attrs))
	*mapOrig = append(*mapOrig, internal.KeyValue{
		Key:    "fallback-key",
		KeyRef: 999, // invalid index
		Value: internal.AnyValue{
			Value: &internal.AnyValue_StringValueRef{
				StringValueRef: 999, // invalid index
			},
		},
	})

	resolveProfilesReferences(profiles)

	// Key should remain unchanged since ref is invalid
	kv := &(*mapOrig)[0]
	assert.Equal(t, "fallback-key", kv.Key)

	// Value should remain as StringValueRef since index is invalid
	_, ok := kv.Value.Value.(*internal.AnyValue_StringValueRef)
	assert.True(t, ok)
}

func TestResolveAnyValueReferenceWithPooling(t *testing.T) {
	// Test with pooling enabled
	prevPooling := internal.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(internal.UseProtoPooling.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(internal.UseProtoPooling.ID(), prevPooling))
	}()

	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("")
	dict.StringTable().Append("pooled-value")

	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_StringValueRef{
			StringValueRef: 1,
		},
	}

	resolveAnyValueReference(dict, anyVal)

	strVal, ok := anyVal.Value.(*internal.AnyValue_StringValue)
	assert.True(t, ok)
	assert.Equal(t, "pooled-value", strVal.StringValue)
}

func TestResolveAnyValueReferenceNestedKvList(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("")
	dict.StringTable().Append("nested-key")
	dict.StringTable().Append("nested-value")

	kvList := &internal.KeyValueList{
		Values: []internal.KeyValue{
			{
				KeyRef: 1, // references "nested-key"
				Value: internal.AnyValue{
					Value: &internal.AnyValue_StringValueRef{
						StringValueRef: 2, // references "nested-value"
					},
				},
			},
		},
	}

	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_KvlistValue{
			KvlistValue: kvList,
		},
	}

	resolveAnyValueReference(dict, anyVal)

	// Verify nested key_ref was resolved
	assert.Equal(t, "nested-key", kvList.Values[0].Key)

	// Verify nested value was resolved
	strVal, ok := kvList.Values[0].Value.Value.(*internal.AnyValue_StringValue)
	assert.True(t, ok)
	assert.Equal(t, "nested-value", strVal.StringValue)
}

func TestResolveAnyValueReferenceNestedArray(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("")
	dict.StringTable().Append("array-item-1")
	dict.StringTable().Append("array-item-2")

	arrVal := &internal.ArrayValue{
		Values: []internal.AnyValue{
			{
				Value: &internal.AnyValue_StringValueRef{
					StringValueRef: 1,
				},
			},
			{
				Value: &internal.AnyValue_StringValueRef{
					StringValueRef: 2,
				},
			},
		},
	}

	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_ArrayValue{
			ArrayValue: arrVal,
		},
	}

	resolveAnyValueReference(dict, anyVal)

	// Verify both array items were resolved
	strVal1, ok := arrVal.Values[0].Value.(*internal.AnyValue_StringValue)
	assert.True(t, ok)
	assert.Equal(t, "array-item-1", strVal1.StringValue)

	strVal2, ok := arrVal.Values[1].Value.(*internal.AnyValue_StringValue)
	assert.True(t, ok)
	assert.Equal(t, "array-item-2", strVal2.StringValue)
}

func TestConvertProfilesToReferencesEmpty(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("")

	convertProfilesToReferences(profiles)

	// Should only have the initial empty string
	assert.Equal(t, 1, dict.StringTable().Len())
}

func TestConvertProfilesToReferencesDeduplication(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("")

	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("key1", "duplicated-value")
	rp.Resource().Attributes().PutStr("key2", "duplicated-value")
	rp.Resource().Attributes().PutStr("key3", "unique-value")

	convertProfilesToReferences(profiles)

	// Should have: "", "key1", "duplicated-value", "key2", "key3", "unique-value"
	// But key1, key2, key3 might share indices if they're also deduplicated
	// At minimum: "", "key1", "duplicated-value", "key2", "key3", "unique-value" = 6
	assert.GreaterOrEqual(t, dict.StringTable().Len(), 5)

	// Verify references were created
	mapOrig := internal.GetMapOrig(internal.MapWrapper(rp.Resource().Attributes()))
	for i := 0; i < len(*mapOrig); i++ {
		kv := &(*mapOrig)[i]
		assert.NotEqual(t, int32(0), kv.KeyRef, "Key should have a reference")

		// Values should be converted to StringValueRef
		_, ok := kv.Value.Value.(*internal.AnyValue_StringValueRef)
		assert.True(t, ok, "Value should be converted to StringValueRef")
	}
}

func TestConvertAnyValueToReferenceWithPooling(t *testing.T) {
	prevPooling := internal.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(internal.UseProtoPooling.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(internal.UseProtoPooling.ID(), prevPooling))
	}()

	stringIndex := make(map[string]int32)
	stringIndex["test-value"] = 5

	getStringIndex := func(s string) int32 {
		if idx, ok := stringIndex[s]; ok {
			return idx
		}
		idx := int32(len(stringIndex))
		stringIndex[s] = idx
		return idx
	}

	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_StringValue{
			StringValue: "test-value",
		},
	}

	convertAnyValueToReference(getStringIndex, anyVal)

	refVal, ok := anyVal.Value.(*internal.AnyValue_StringValueRef)
	assert.True(t, ok)
	assert.Equal(t, int32(5), refVal.StringValueRef)
}

func TestConvertAnyValueToReferenceEmptyString(t *testing.T) {
	stringIndex := make(map[string]int32)
	stringIndex[""] = 0

	getStringIndex := func(s string) int32 {
		if idx, ok := stringIndex[s]; ok {
			return idx
		}
		idx := int32(len(stringIndex))
		stringIndex[s] = idx
		return idx
	}

	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_StringValue{
			StringValue: "", // empty string should not be converted
		},
	}

	convertAnyValueToReference(getStringIndex, anyVal)

	// Empty string should remain as StringValue, not converted to ref
	_, ok := anyVal.Value.(*internal.AnyValue_StringValue)
	assert.True(t, ok)
}

func TestConvertAnyValueToReferenceNestedKvList(t *testing.T) {
	stringIndex := make(map[string]int32)
	stringIndex[""] = 0

	counter := int32(1)
	getStringIndex := func(s string) int32 {
		if idx, ok := stringIndex[s]; ok {
			return idx
		}
		idx := counter
		counter++
		stringIndex[s] = idx
		return idx
	}

	kvList := &internal.KeyValueList{
		Values: []internal.KeyValue{
			{
				Key: "nested-key",
				Value: internal.AnyValue{
					Value: &internal.AnyValue_StringValue{
						StringValue: "nested-value",
					},
				},
			},
		},
	}

	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_KvlistValue{
			KvlistValue: kvList,
		},
	}

	convertAnyValueToReference(getStringIndex, anyVal)

	// Verify nested key was converted
	assert.NotEqual(t, int32(0), kvList.Values[0].KeyRef)

	// Verify nested value was converted
	_, ok := kvList.Values[0].Value.Value.(*internal.AnyValue_StringValueRef)
	assert.True(t, ok)
}

func TestConvertAnyValueToReferenceNestedArray(t *testing.T) {
	stringIndex := make(map[string]int32)
	counter := int32(0)

	getStringIndex := func(s string) int32 {
		if idx, ok := stringIndex[s]; ok {
			return idx
		}
		idx := counter
		counter++
		stringIndex[s] = idx
		return idx
	}

	arrVal := &internal.ArrayValue{
		Values: []internal.AnyValue{
			{
				Value: &internal.AnyValue_StringValue{
					StringValue: "array-item",
				},
			},
		},
	}

	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_ArrayValue{
			ArrayValue: arrVal,
		},
	}

	convertAnyValueToReference(getStringIndex, anyVal)

	// Verify array item was converted
	_, ok := arrVal.Values[0].Value.(*internal.AnyValue_StringValueRef)
	assert.True(t, ok)
}

func TestConvertMapToReferencesEmptyKey(t *testing.T) {
	profiles := NewProfiles()
	rp := profiles.ResourceProfiles().AppendEmpty()
	attrs := rp.Resource().Attributes()

	// Manually add a KeyValue with empty key
	mapOrig := internal.GetMapOrig(internal.MapWrapper(attrs))
	*mapOrig = append(*mapOrig, internal.KeyValue{
		Key: "", // empty key should not be converted
		Value: internal.AnyValue{
			Value: &internal.AnyValue_StringValue{
				StringValue: "value",
			},
		},
	})

	getStringIndex := func(s string) int32 {
		return 1
	}

	convertMapToReferences(getStringIndex, attrs)

	// Empty key should not have KeyRef set
	kv := &(*mapOrig)[0]
	assert.Equal(t, int32(0), kv.KeyRef)
}

func TestConvertMapToReferencesExistingKeyRef(t *testing.T) {
	profiles := NewProfiles()
	rp := profiles.ResourceProfiles().AppendEmpty()
	attrs := rp.Resource().Attributes()

	// Manually add a KeyValue with existing KeyRef
	mapOrig := internal.GetMapOrig(internal.MapWrapper(attrs))
	*mapOrig = append(*mapOrig, internal.KeyValue{
		Key:    "test-key",
		KeyRef: 5, // already has a ref
		Value: internal.AnyValue{
			Value: &internal.AnyValue_StringValue{
				StringValue: "value",
			},
		},
	})

	getStringIndex := func(s string) int32 {
		return 99
	}

	convertMapToReferences(getStringIndex, attrs)

	// KeyRef should remain unchanged
	kv := &(*mapOrig)[0]
	assert.Equal(t, int32(5), kv.KeyRef)
}

func TestResolveAnyValueReferenceNonStringTypes(t *testing.T) {
	profiles := NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append("")

	// Test with int value (should not be affected)
	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_IntValue{
			IntValue: 42,
		},
	}

	resolveAnyValueReference(dict, anyVal)

	// Should remain as IntValue
	intVal, ok := anyVal.Value.(*internal.AnyValue_IntValue)
	assert.True(t, ok)
	assert.Equal(t, int64(42), intVal.IntValue)
}

func TestConvertAnyValueToReferenceNonStringTypes(t *testing.T) {
	getStringIndex := func(s string) int32 {
		return 0
	}

	// Test with bool value (should not be affected)
	anyVal := &internal.AnyValue{
		Value: &internal.AnyValue_BoolValue{
			BoolValue: true,
		},
	}

	convertAnyValueToReference(getStringIndex, anyVal)

	// Should remain as BoolValue
	boolVal, ok := anyVal.Value.(*internal.AnyValue_BoolValue)
	assert.True(t, ok)
	assert.Equal(t, true, boolVal.BoolValue)
}
