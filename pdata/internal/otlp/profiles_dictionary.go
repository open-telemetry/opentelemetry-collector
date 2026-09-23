// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

// ConvertProfilesToReferences interns resource and scope attribute strings in
// the profiles dictionary. The request must be mutable.
func ConvertProfilesToReferences(request *internal.ExportProfilesServiceRequest) {
	stringTable := &request.Dictionary.StringTable

	var stringIndex map[string]int32
	getStringIndex := func(s string) int32 {
		if stringIndex == nil {
			if len(*stringTable) == 0 {
				*stringTable = append(*stringTable, "")
			}
			stringIndex = make(map[string]int32, len(*stringTable))
			for i, value := range *stringTable {
				stringIndex[value] = int32(i)
			}
		}
		if index, ok := stringIndex[s]; ok {
			return index
		}
		index := int32(len(*stringTable))
		*stringTable = append(*stringTable, s)
		stringIndex[s] = index
		return index
	}

	for _, resourceProfiles := range request.ResourceProfiles {
		ConvertProfilesKeyValuesToReferences(getStringIndex, resourceProfiles.Resource.Attributes)
		for _, scopeProfiles := range resourceProfiles.ScopeProfiles {
			ConvertProfilesKeyValuesToReferences(getStringIndex, scopeProfiles.Scope.Attributes)
		}
	}
}

// ResolveProfilesReferences resolves resource and scope attribute string-table
// references so pdata consumers can access the attributes transparently.
func ResolveProfilesReferences(request *internal.ExportProfilesServiceRequest) {
	stringTable := request.Dictionary.StringTable
	for _, resourceProfiles := range request.ResourceProfiles {
		ResolveProfilesKeyValueReferences(stringTable, resourceProfiles.Resource.Attributes)
		for _, scopeProfiles := range resourceProfiles.ScopeProfiles {
			ResolveProfilesKeyValueReferences(stringTable, scopeProfiles.Scope.Attributes)
		}
	}
}

// ConvertProfilesKeyValuesToReferences converts attribute strings to dictionary references.
func ConvertProfilesKeyValuesToReferences(getStringIndex func(string) int32, keyValues []internal.KeyValue) {
	for i := range keyValues {
		keyValue := &keyValues[i]
		if keyValue.Key != "" {
			keyValue.KeyStrindex = getStringIndex(keyValue.Key)
			keyValue.Key = ""
		}
		ConvertProfilesAnyValueToReference(getStringIndex, &keyValue.Value)
	}
}

// ConvertProfilesAnyValueToReference converts strings recursively in an attribute value.
func ConvertProfilesAnyValueToReference(getStringIndex func(string) int32, value *internal.AnyValue) {
	if _, ok := value.Value.(*internal.AnyValue_StringValueStrindex); ok {
		return
	}

	switch original := value.Value.(type) {
	case *internal.AnyValue_StringValue:
		if original.StringValue == "" {
			return
		}
		var reference *internal.AnyValue_StringValueStrindex
		if metadata.PdataUseProtoPoolingFeatureGate.IsEnabled() {
			reference = internal.ProtoPoolAnyValue_StringValueStrindex.Get().(*internal.AnyValue_StringValueStrindex)
		} else {
			reference = &internal.AnyValue_StringValueStrindex{}
		}
		reference.StringValueStrindex = getStringIndex(original.StringValue)
		value.Value = reference
	case *internal.AnyValue_KvlistValue:
		if original.KvlistValue != nil {
			ConvertProfilesKeyValuesToReferences(getStringIndex, original.KvlistValue.Values)
		}
	case *internal.AnyValue_ArrayValue:
		if original.ArrayValue != nil {
			for i := range original.ArrayValue.Values {
				ConvertProfilesAnyValueToReference(getStringIndex, &original.ArrayValue.Values[i])
			}
		}
	}
}

// ResolveProfilesKeyValueReferences resolves attribute dictionary references.
func ResolveProfilesKeyValueReferences(stringTable []string, keyValues []internal.KeyValue) {
	for i := range keyValues {
		keyValue := &keyValues[i]
		if keyValue.KeyStrindex > 0 && keyValue.Key == "" && int(keyValue.KeyStrindex) < len(stringTable) {
			keyValue.Key = stringTable[keyValue.KeyStrindex]
			keyValue.KeyStrindex = 0
		}
		ResolveProfilesAnyValueReference(stringTable, &keyValue.Value)
	}
}

// ResolveProfilesAnyValueReference resolves string references recursively in an attribute value.
func ResolveProfilesAnyValueReference(stringTable []string, value *internal.AnyValue) {
	switch original := value.Value.(type) {
	case *internal.AnyValue_StringValueStrindex:
		if original.StringValueStrindex <= 0 || int(original.StringValueStrindex) >= len(stringTable) {
			return
		}
		var resolved *internal.AnyValue_StringValue
		if metadata.PdataUseProtoPoolingFeatureGate.IsEnabled() {
			resolved = internal.ProtoPoolAnyValue_StringValue.Get().(*internal.AnyValue_StringValue)
		} else {
			resolved = &internal.AnyValue_StringValue{}
		}
		resolved.StringValue = stringTable[original.StringValueStrindex]
		value.Value = resolved
	case *internal.AnyValue_KvlistValue:
		if original.KvlistValue != nil {
			ResolveProfilesKeyValueReferences(stringTable, original.KvlistValue.Values)
		}
	case *internal.AnyValue_ArrayValue:
		if original.ArrayValue != nil {
			for i := range original.ArrayValue.Values {
				ResolveProfilesAnyValueReference(stringTable, &original.ArrayValue.Values[i])
			}
		}
	}
}
