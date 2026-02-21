// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// resolveProfilesReferences walks through all profiles data after unmarshaling
// and resolves any string_value_ref and key_ref to their actual string values.
// This ensures the pdata API works transparently with referenced strings.
func resolveProfilesReferences(profiles Profiles) {
	dict := profiles.Dictionary()

	// Resolve references in resource attributes
	for i := 0; i < profiles.ResourceProfiles().Len(); i++ {
		rp := profiles.ResourceProfiles().At(i)
		resolveMapReferences(dict, rp.Resource().Attributes())

		// Resolve references in scope attributes
		for j := 0; j < rp.ScopeProfiles().Len(); j++ {
			sp := rp.ScopeProfiles().At(j)
			resolveMapReferences(dict, sp.Scope().Attributes())
		}
	}
}

// resolveMapReferences resolves all string_value_ref and key_ref in a map
func resolveMapReferences(dict ProfilesDictionary, m pcommon.Map) {
	mapOrig := internal.GetMapOrig(internal.MapWrapper(m))

	for i := 0; i < len(*mapOrig); i++ {
		kv := &(*mapOrig)[i]

		// Resolve key_ref if set
		if kv.KeyRef >= 0 {
			idx := int(kv.KeyRef)
			if idx < dict.StringTable().Len() {
				kv.Key = dict.StringTable().At(idx)
				// Keep ref set for potential re-marshaling
			}
		}

		// Resolve string_value_ref if set
		resolveAnyValueReference(dict, &kv.Value)
	}
}

// resolveAnyValueReference resolves string_value_ref in an AnyValue
func resolveAnyValueReference(dict ProfilesDictionary, anyValue *internal.AnyValue) {
	if ref, ok := anyValue.Value.(*internal.AnyValue_StringValueRef); ok && ref.StringValueRef != 0 {
		idx := int(ref.StringValueRef)
		if idx >= 0 && idx < dict.StringTable().Len() {
			str := dict.StringTable().At(idx)
			var ov *internal.AnyValue_StringValue
			if !internal.UseProtoPooling.IsEnabled() {
				ov = &internal.AnyValue_StringValue{}
			} else {
				ov = internal.ProtoPoolAnyValue_StringValue.Get().(*internal.AnyValue_StringValue)
			}
			ov.StringValue = str
			anyValue.Value = ov
		}
	} else if kvList, ok := anyValue.Value.(*internal.AnyValue_KvlistValue); ok && kvList.KvlistValue != nil {
		for i := 0; i < len(kvList.KvlistValue.Values); i++ {
			kv := &kvList.KvlistValue.Values[i]
			if kv.KeyRef >= 0 {
				idx := int(kv.KeyRef)
				if idx < dict.StringTable().Len() {
					kv.Key = dict.StringTable().At(idx)
				}
			}
			resolveAnyValueReference(dict, &kv.Value)
		}
	} else if arrVal, ok := anyValue.Value.(*internal.AnyValue_ArrayValue); ok && arrVal.ArrayValue != nil {
		for i := 0; i < len(arrVal.ArrayValue.Values); i++ {
			resolveAnyValueReference(dict, &arrVal.ArrayValue.Values[i])
		}
	}
}

// convertProfilesToReferences walks through all profiles data before marshaling
// and converts string values to references for efficient transmission.
// This builds up the string table in the dictionary and replaces strings with refs.
func convertProfilesToReferences(profiles Profiles) {
	dict := profiles.Dictionary()
	stringTable := dict.StringTable()

	// Map for quick string lookups - only allocate if needed
	var stringIndex map[string]int32
	getStringIndex := func(s string) int32 {
		if stringIndex == nil {
			stringIndex = make(map[string]int32, stringTable.Len())
			for i := 0; i < stringTable.Len(); i++ {
				stringIndex[stringTable.At(i)] = int32(i)
			}
		}

		if idx, ok := stringIndex[s]; ok {
			return idx
		}
		idx := int32(stringTable.Len())
		stringTable.Append(s)
		stringIndex[s] = idx
		return idx
	}

	// Convert strings in resource attributes
	for i := 0; i < profiles.ResourceProfiles().Len(); i++ {
		rp := profiles.ResourceProfiles().At(i)
		convertMapToReferences(getStringIndex, rp.Resource().Attributes())

		// Convert strings in scope attributes
		for j := 0; j < rp.ScopeProfiles().Len(); j++ {
			sp := rp.ScopeProfiles().At(j)
			convertMapToReferences(getStringIndex, sp.Scope().Attributes())
		}
	}
}

// convertMapToReferences converts string keys and values to references
func convertMapToReferences(getStringIndex func(string) int32, m pcommon.Map) {
	mapOrig := internal.GetMapOrig(internal.MapWrapper(m))

	for i := 0; i < len(*mapOrig); i++ {
		kv := &(*mapOrig)[i]

		// Convert key to reference
		if kv.Key != "" && kv.KeyRef == 0 {
			kv.KeyRef = getStringIndex(kv.Key)
			kv.Key = ""
		}

		// Convert string values to references
		convertAnyValueToReference(getStringIndex, &kv.Value)
	}
}

// convertAnyValueToReference converts string values to string_value_ref
func convertAnyValueToReference(getStringIndex func(string) int32, anyValue *internal.AnyValue) {
	// Skip if already a reference
	if _, ok := anyValue.Value.(*internal.AnyValue_StringValueRef); ok {
		return
	}

	if strVal, ok := anyValue.Value.(*internal.AnyValue_StringValue); ok && strVal.StringValue != "" {
		// Convert to reference
		idx := getStringIndex(strVal.StringValue)
		var ov *internal.AnyValue_StringValueRef
		if !internal.UseProtoPooling.IsEnabled() {
			ov = &internal.AnyValue_StringValueRef{}
		} else {
			ov = internal.ProtoPoolAnyValue_StringValueRef.Get().(*internal.AnyValue_StringValueRef)
		}
		ov.StringValueRef = idx
		anyValue.Value = ov
	} else if kvList, ok := anyValue.Value.(*internal.AnyValue_KvlistValue); ok && kvList.KvlistValue != nil {
		// Recursively convert nested key-value lists
		for i := 0; i < len(kvList.KvlistValue.Values); i++ {
			kv := &kvList.KvlistValue.Values[i]
			if kv.Key != "" && kv.KeyRef == 0 {
				kv.KeyRef = getStringIndex(kv.Key)
				kv.Key = ""
			}
			convertAnyValueToReference(getStringIndex, &kv.Value)
		}
	} else if arrVal, ok := anyValue.Value.(*internal.AnyValue_ArrayValue); ok && arrVal.ArrayValue != nil {
		// Recursively convert arrays
		for i := 0; i < len(arrVal.ArrayValue.Values); i++ {
			convertAnyValueToReference(getStringIndex, &arrVal.ArrayValue.Values[i])
		}
	}
}
