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

	// Quick check: if there are no resource profiles, nothing to do
	if profiles.ResourceProfiles().Len() == 0 {
		return
	}

	// Check if resolution is needed by sampling first resource
	rp := profiles.ResourceProfiles().At(0)
	if !needsResolution(rp.Resource().Attributes()) {
		// Check scope attributes too
		if rp.ScopeProfiles().Len() == 0 {
			return
		}
		sp := rp.ScopeProfiles().At(0)
		if !needsResolution(sp.Scope().Attributes()) {
			// Already resolved, skip
			return
		}
	}

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

// needsResolution checks if a map has any refs that need resolution
func needsResolution(m pcommon.Map) bool {
	if m.Len() == 0 {
		return false
	}
	mapOrig := internal.GetMapOrig(internal.MapWrapper(m))
	for i := 0; i < len(*mapOrig); i++ {
		kv := &(*mapOrig)[i]
		// If KeyRef is set, needs resolution
		if kv.KeyRef != 0 {
			return true
		}
		// Check if any values need resolution
		if anyValueNeedsResolution(&kv.Value) {
			return true
		}
	}
	return false
}

// anyValueNeedsResolution checks if an AnyValue has refs that need resolution
func anyValueNeedsResolution(anyValue *internal.AnyValue) bool {
	if ref, ok := anyValue.Value.(*internal.AnyValue_StringValueRef); ok && ref.StringValueRef != 0 {
		return true
	} else if kvList, ok := anyValue.Value.(*internal.AnyValue_KvlistValue); ok && kvList.KvlistValue != nil {
		for i := 0; i < len(kvList.KvlistValue.Values); i++ {
			kv := &kvList.KvlistValue.Values[i]
			if kv.KeyRef != 0 {
				return true
			}
			if anyValueNeedsResolution(&kv.Value) {
				return true
			}
		}
	} else if arrVal, ok := anyValue.Value.(*internal.AnyValue_ArrayValue); ok && arrVal.ArrayValue != nil {
		for i := 0; i < len(arrVal.ArrayValue.Values); i++ {
			if anyValueNeedsResolution(&arrVal.ArrayValue.Values[i]) {
				return true
			}
		}
	}
	return false
}

// resolveMapReferences resolves all string_value_ref and key_ref in a map
func resolveMapReferences(dict ProfilesDictionary, m pcommon.Map) {
	mapOrig := internal.GetMapOrig(internal.MapWrapper(m))

	for i := 0; i < len(*mapOrig); i++ {
		kv := &(*mapOrig)[i]

		// Resolve key_ref if set
		if kv.KeyRef != 0 {
			idx := int(kv.KeyRef)
			if idx >= 0 && idx < dict.StringTable().Len() {
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
			if kv.KeyRef != 0 {
				idx := int(kv.KeyRef)
				if idx >= 0 && idx < dict.StringTable().Len() {
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

	// Quick check: if there are no resource profiles, nothing to do
	if profiles.ResourceProfiles().Len() == 0 {
		return
	}

	// Check if conversion is needed by sampling first resource
	rp := profiles.ResourceProfiles().At(0)
	if !needsConversion(rp.Resource().Attributes()) {
		// Check scope attributes too
		if rp.ScopeProfiles().Len() == 0 {
			return
		}
		sp := rp.ScopeProfiles().At(0)
		if !needsConversion(sp.Scope().Attributes()) {
			// Already converted, skip
			return
		}
	}

	// Map for quick string lookups - only allocate if needed
	stringIndex := make(map[string]int32, stringTable.Len())
	for i := 0; i < stringTable.Len(); i++ {
		stringIndex[stringTable.At(i)] = int32(i)
	}

	getStringIndex := func(s string) int32 {
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

// needsConversion checks if a map has any string values that need conversion to refs
func needsConversion(m pcommon.Map) bool {
	if m.Len() == 0 {
		return false
	}
	mapOrig := internal.GetMapOrig(internal.MapWrapper(m))
	for i := 0; i < len(*mapOrig); i++ {
		kv := &(*mapOrig)[i]
		// If KeyRef is not set but Key is, needs conversion
		if kv.Key != "" && kv.KeyRef == 0 {
			return true
		}
		// Check if any string values need conversion
		if anyValueNeedsConversion(&kv.Value) {
			return true
		}
	}
	return false
}

// anyValueNeedsConversion checks if an AnyValue has string values that need conversion
func anyValueNeedsConversion(anyValue *internal.AnyValue) bool {
	if strVal, ok := anyValue.Value.(*internal.AnyValue_StringValue); ok && strVal.StringValue != "" {
		return true
	} else if kvList, ok := anyValue.Value.(*internal.AnyValue_KvlistValue); ok && kvList.KvlistValue != nil {
		for i := 0; i < len(kvList.KvlistValue.Values); i++ {
			kv := &kvList.KvlistValue.Values[i]
			if kv.Key != "" && kv.KeyRef == 0 {
				return true
			}
			if anyValueNeedsConversion(&kv.Value) {
				return true
			}
		}
	} else if arrVal, ok := anyValue.Value.(*internal.AnyValue_ArrayValue); ok && arrVal.ArrayValue != nil {
		for i := 0; i < len(arrVal.ArrayValue.Values); i++ {
			if anyValueNeedsConversion(&arrVal.ArrayValue.Values[i]) {
				return true
			}
		}
	}
	return false
}

// convertMapToReferences converts string keys and values to references
func convertMapToReferences(getStringIndex func(string) int32, m pcommon.Map) {
	mapOrig := internal.GetMapOrig(internal.MapWrapper(m))

	for i := 0; i < len(*mapOrig); i++ {
		kv := &(*mapOrig)[i]

		// Convert key to reference
		if kv.Key != "" && kv.KeyRef == 0 {
			kv.KeyRef = getStringIndex(kv.Key)
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
