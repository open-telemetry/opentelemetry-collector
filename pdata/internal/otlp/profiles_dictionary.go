// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp // import "go.opentelemetry.io/collector/pdata/internal/otlp"

import (
	"errors"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

var (
	errProfilesStringIndexNotInitialized = errors.New("profiles dictionary string index is not initialized")
	errProfilesStringTableTooLarge       = errors.New("profiles dictionary string table has too many entries")
)

// ConvertProfilesToReferences interns resource and scope attribute strings in
// the profiles dictionary. The request must be mutable.
func ConvertProfilesToReferences(request *internal.ExportProfilesServiceRequest) error {
	stringTable := &request.Dictionary.StringTable
	if len(*stringTable) == 0 {
		// string_table[0] is the required empty-string sentinel.
		*stringTable = append(*stringTable, "")
	} else if (*stringTable)[0] != "" {
		return errors.New("profiles dictionary string_table[0] must be empty")
	}
	if len(*stringTable) > math.MaxInt32 {
		return errProfilesStringTableTooLarge
	}

	stringIndex := make(map[string]int32, len(*stringTable))
	for i, value := range *stringTable {
		// Keep the first occurrence so the required empty string resolves to
		// the canonical index 0 even if the table already contains duplicates.
		if _, ok := stringIndex[value]; !ok {
			stringIndex[value] = int32(i)
		}
	}

	getStringIndex := func(s string) (int32, error) {
		if stringIndex == nil {
			return 0, errProfilesStringIndexNotInitialized
		}
		if index, ok := stringIndex[s]; ok {
			return index, nil
		}
		if len(*stringTable) >= math.MaxInt32 {
			return 0, errProfilesStringTableTooLarge
		}
		index := int32(len(*stringTable))
		*stringTable = append(*stringTable, s)
		stringIndex[s] = index
		return index, nil
	}

	for resourceIndex, resourceProfiles := range request.ResourceProfiles {
		if err := ConvertProfilesKeyValuesToReferences(getStringIndex, resourceProfiles.Resource.Attributes); err != nil {
			return fmt.Errorf("resource profiles %d resource attributes: %w", resourceIndex, err)
		}
		for scopeIndex, scopeProfiles := range resourceProfiles.ScopeProfiles {
			if err := ConvertProfilesKeyValuesToReferences(getStringIndex, scopeProfiles.Scope.Attributes); err != nil {
				return fmt.Errorf("resource profiles %d scope profiles %d attributes: %w", resourceIndex, scopeIndex, err)
			}
		}
	}
	return nil
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
func ConvertProfilesKeyValuesToReferences(getStringIndex func(string) (int32, error), keyValues []internal.KeyValue) error {
	for i := range keyValues {
		keyValue := &keyValues[i]
		if keyValue.Key != "" && keyValue.KeyStrindex != 0 {
			return fmt.Errorf("attribute %d has both key and key_strindex set", i)
		}
		if keyValue.Key != "" {
			index, err := getStringIndex(keyValue.Key)
			if err != nil {
				return err
			}
			keyValue.KeyStrindex = index
			keyValue.Key = ""
		}
		if err := ConvertProfilesAnyValueToReference(getStringIndex, &keyValue.Value); err != nil {
			return err
		}
	}
	return nil
}

// ConvertProfilesAnyValueToReference converts strings recursively in an attribute value.
func ConvertProfilesAnyValueToReference(getStringIndex func(string) (int32, error), value *internal.AnyValue) error {
	if _, ok := value.Value.(*internal.AnyValue_StringValueStrindex); ok {
		return nil
	}

	switch original := value.Value.(type) {
	case *internal.AnyValue_StringValue:
		index, err := getStringIndex(original.StringValue)
		if err != nil {
			return err
		}
		var reference *internal.AnyValue_StringValueStrindex
		if metadata.PdataUseProtoPoolingFeatureGate.IsEnabled() {
			reference = internal.ProtoPoolAnyValue_StringValueStrindex.Get().(*internal.AnyValue_StringValueStrindex)
		} else {
			reference = &internal.AnyValue_StringValueStrindex{}
		}
		reference.StringValueStrindex = index
		value.Value = reference
	case *internal.AnyValue_KvlistValue:
		if original.KvlistValue != nil {
			return ConvertProfilesKeyValuesToReferences(getStringIndex, original.KvlistValue.Values)
		}
	case *internal.AnyValue_ArrayValue:
		if original.ArrayValue != nil {
			for i := range original.ArrayValue.Values {
				if err := ConvertProfilesAnyValueToReference(getStringIndex, &original.ArrayValue.Values[i]); err != nil {
					return err
				}
			}
		}
	}
	return nil
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
		if original.StringValueStrindex < 0 || int(original.StringValueStrindex) >= len(stringTable) {
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
