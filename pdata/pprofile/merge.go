// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"
)

func safeInt32(i int) (int32, error) {
	if i < 0 || i > math.MaxInt32 {
		return 0, fmt.Errorf("value %d outside int32 range", i)
	}
	// Safe to convert since we've checked the range
	return int32(i), nil
}

var (
	ErrDictionarySizeExceeded = fmt.Errorf("dictionary size would exceed maximum limit of %d entries", math.MaxInt32)
	ErrSourceReadOnly         = errors.New("cannot merge from read-only source")
	ErrDestinationReadOnly    = errors.New("cannot merge into read-only destination")
)

// MergeFrom merges the profiles from the source Profiles into the current Profiles instance.
// This operation deduplicates dictionary entries and updates all references accordingly.
//
// The merge process:
// 1. Merges all dictionary tables (strings, functions, mappings, locations, links, attributes)
// 2. Deduplicates entries based on equality and builds index mappings for reference updates
// 3. Updates all references in resource profiles to point to the merged dictionary entries
// 4. Appends all resource profiles from source to destination
//
// Constraints:
// - Both source and destination must be mutable (not read-only)
// - Dictionary tables cannot exceed math.MaxInt32 entries after merging
// - Profile IDs are preserved from the source profiles
func (ms Profiles) MergeFrom(source Profiles) error {
	if ms.IsReadOnly() {
		return ErrDestinationReadOnly
	}
	if source.IsReadOnly() {
		return ErrSourceReadOnly
	}

	destDict := ms.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	indexMappings, err := ms.mergeDictionaries(destDict, sourceDict)
	if err != nil {
		return err
	}

	return ms.mergeResourceProfiles(source.ResourceProfiles(), indexMappings)
}

type indexMappings struct {
	stringTable    map[int32]int32
	functionTable  map[int32]int32
	mappingTable   map[int32]int32
	locationTable  map[int32]int32
	linkTable      map[int32]int32
	attributeTable map[int32]int32
}

// mergeDictionaries merges all dictionary tables from source to destination and returns index mappings
func (ms Profiles) mergeDictionaries(destDict, sourceDict ProfilesDictionary) (*indexMappings, error) {
	mappings := &indexMappings{
		stringTable:    make(map[int32]int32),
		functionTable:  make(map[int32]int32),
		mappingTable:   make(map[int32]int32),
		locationTable:  make(map[int32]int32),
		linkTable:      make(map[int32]int32),
		attributeTable: make(map[int32]int32),
	}

	if err := ms.checkSizeLimits(destDict, sourceDict); err != nil {
		return nil, err
	}

	if err := ms.mergeStringTable(destDict, sourceDict, mappings.stringTable); err != nil {
		return nil, err
	}

	if err := ms.mergeAttributeTable(destDict, sourceDict, mappings.attributeTable); err != nil {
		return nil, err
	}
	ms.mergeAttributeUnits(destDict, sourceDict, mappings.stringTable)
	if err := ms.mergeFunctionTable(destDict, sourceDict, mappings.functionTable, mappings.stringTable); err != nil {
		return nil, err
	}
	if err := ms.mergeMappingTable(destDict, sourceDict, mappings.mappingTable, mappings.stringTable); err != nil {
		return nil, err
	}
	if err := ms.mergeLocationTable(destDict, sourceDict, mappings.locationTable, mappings.mappingTable, mappings.functionTable); err != nil {
		return nil, err
	}
	if err := ms.mergeLinkTable(destDict, sourceDict, mappings.linkTable); err != nil {
		return nil, err
	}

	return mappings, nil
}

// checkSizeLimits validates that merging dictionaries won't exceed size limits
func (ms Profiles) checkSizeLimits(destDict, sourceDict ProfilesDictionary) error {
	if int64(destDict.StringTable().Len())+int64(sourceDict.StringTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.FunctionTable().Len())+int64(sourceDict.FunctionTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.MappingTable().Len())+int64(sourceDict.MappingTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.LocationTable().Len())+int64(sourceDict.LocationTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.LinkTable().Len())+int64(sourceDict.LinkTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.AttributeTable().Len())+int64(sourceDict.AttributeTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.AttributeUnits().Len())+int64(sourceDict.AttributeUnits().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	return nil
}

// mergeStringTable merges string tables and builds index mapping
func (ms Profiles) mergeStringTable(destDict, sourceDict ProfilesDictionary, stringMapping map[int32]int32) error {
	destStrings := destDict.StringTable()
	sourceStrings := sourceDict.StringTable()

	for i := range sourceStrings.Len() {
		sourceStr := sourceStrings.At(i)
		newIndex, err := SetString(destStrings, sourceStr)
		if err != nil {
			return fmt.Errorf("failed to merge string table entry at index %d: %w", i, err)
		}
		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("string table index: %w", err)
		}
		stringMapping[idx] = newIndex
	}

	return nil
}

// mergeFunctionTable merges function tables and builds index mapping
func (ms Profiles) mergeFunctionTable(destDict, sourceDict ProfilesDictionary, functionMapping, stringMapping map[int32]int32) error {
	destFunctions := destDict.FunctionTable()
	sourceFunctions := sourceDict.FunctionTable()

	for i := range sourceFunctions.Len() {
		sourceFunc := sourceFunctions.At(i)

		updatedFunc := NewFunction()
		sourceFunc.CopyTo(updatedFunc)

		if nameIdx := updatedFunc.NameStrindex(); nameIdx >= 0 {
			if newIdx, exists := stringMapping[nameIdx]; exists {
				updatedFunc.SetNameStrindex(newIdx)
			}
		}
		if sysNameIdx := updatedFunc.SystemNameStrindex(); sysNameIdx >= 0 {
			if newIdx, exists := stringMapping[sysNameIdx]; exists {
				updatedFunc.SetSystemNameStrindex(newIdx)
			}
		}
		if filenameIdx := updatedFunc.FilenameStrindex(); filenameIdx >= 0 {
			if newIdx, exists := stringMapping[filenameIdx]; exists {
				updatedFunc.SetFilenameStrindex(newIdx)
			}
		}

		var newIndex int32
		found := false
		for j, existingFunc := range destFunctions.All() {
			if !existingFunc.Equal(updatedFunc) {
				continue
			}
			idx, err := safeInt32(j)
			if err != nil {
				return fmt.Errorf("function table index: %w", err)
			}
			newIndex = idx
			found = true
			break
		}

		if !found {
			updatedFunc.CopyTo(destFunctions.AppendEmpty())
			length := destFunctions.Len() - 1
			idx, err := safeInt32(length)
			if err != nil {
				return fmt.Errorf("function table length: %w", err)
			}
			newIndex = idx
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("function mapping index: %w", err)
		}
		functionMapping[idx] = newIndex
	}
	return nil
}

// mergeMappingTable merges mapping tables and builds index mapping
func (ms Profiles) mergeMappingTable(destDict, sourceDict ProfilesDictionary, mappingMapping, stringMapping map[int32]int32) error {
	destMappings := destDict.MappingTable()
	sourceMappings := sourceDict.MappingTable()

	for i := range sourceMappings.Len() {
		sourceMapping := sourceMappings.At(i)

		updatedMapping := NewMapping()
		sourceMapping.CopyTo(updatedMapping)

		if filenameIdx := updatedMapping.FilenameStrindex(); filenameIdx >= 0 {
			if newIdx, exists := stringMapping[filenameIdx]; exists {
				updatedMapping.SetFilenameStrindex(newIdx)
			}
		}

		var newIndex int32
		found := false
		for j, existingMapping := range destMappings.All() {
			if !existingMapping.Equal(updatedMapping) {
				continue
			}
			idx, err := safeInt32(j)
			if err != nil {
				return fmt.Errorf("mapping table index: %w", err)
			}
			newIndex = idx
			found = true
			break
		}

		if !found {
			updatedMapping.CopyTo(destMappings.AppendEmpty())
			length := destMappings.Len() - 1
			idx, err := safeInt32(length)
			if err != nil {
				return fmt.Errorf("mapping table length: %w", err)
			}
			newIndex = idx
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("mapping index: %w", err)
		}
		mappingMapping[idx] = newIndex
	}
	return nil
}

// mergeLocationTable merges location tables and builds index mapping
func (ms Profiles) mergeLocationTable(destDict, sourceDict ProfilesDictionary, locationMapping, mappingMapping, functionMapping map[int32]int32) error {
	destLocations := destDict.LocationTable()
	sourceLocations := sourceDict.LocationTable()

	for i := range sourceLocations.Len() {
		sourceLocation := sourceLocations.At(i)

		updatedLocation := NewLocation()
		sourceLocation.CopyTo(updatedLocation)

		if mappingIdx := updatedLocation.MappingIndex(); mappingIdx >= 0 {
			if newIdx, exists := mappingMapping[mappingIdx]; exists {
				updatedLocation.SetMappingIndex(newIdx)
			}
		}

		lines := updatedLocation.Line()
		for j := range lines.Len() {
			line := lines.At(j)
			if funcIdx := line.FunctionIndex(); funcIdx >= 0 {
				if newIdx, exists := functionMapping[funcIdx]; exists {
					line.SetFunctionIndex(newIdx)
				}
			}
		}

		var newIndex int32
		found := false
		for j, existingLocation := range destLocations.All() {
			if !existingLocation.Equal(updatedLocation) {
				continue
			}
			idx, err := safeInt32(j)
			if err != nil {
				return fmt.Errorf("location table index: %w", err)
			}
			newIndex = idx
			found = true
			break
		}

		if !found {
			updatedLocation.CopyTo(destLocations.AppendEmpty())
			length := destLocations.Len() - 1
			idx, err := safeInt32(length)
			if err != nil {
				return fmt.Errorf("location table length: %w", err)
			}
			newIndex = idx
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("location index: %w", err)
		}
		locationMapping[idx] = newIndex
	}
	return nil
}

// mergeLinkTable merges link tables and builds index mapping
func (ms Profiles) mergeLinkTable(destDict, sourceDict ProfilesDictionary, linkMapping map[int32]int32) error {
	destLinks := destDict.LinkTable()
	sourceLinks := sourceDict.LinkTable()

	for i := range sourceLinks.Len() {
		sourceLink := sourceLinks.At(i)

		var newIndex int32
		found := false
		for j, existingLink := range destLinks.All() {
			if !existingLink.Equal(sourceLink) {
				continue
			}
			idx, err := safeInt32(j)
			if err != nil {
				return fmt.Errorf("link table index: %w", err)
			}
			newIndex = idx
			found = true
			break
		}

		if !found {
			sourceLink.CopyTo(destLinks.AppendEmpty())
			length := destLinks.Len() - 1
			idx, err := safeInt32(length)
			if err != nil {
				return fmt.Errorf("link table length: %w", err)
			}
			newIndex = idx
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("link index: %w", err)
		}
		linkMapping[idx] = newIndex
	}
	return nil
}

// mergeAttributeTable merges attribute tables and builds index mapping
func (ms Profiles) mergeAttributeTable(destDict, sourceDict ProfilesDictionary, attributeMapping map[int32]int32) error {
	destAttributes := destDict.AttributeTable()
	sourceAttributes := sourceDict.AttributeTable()

	for i := range sourceAttributes.Len() {
		sourceAttr := sourceAttributes.At(i)

		var newIndex int32
		found := false
		for j, existingAttr := range destAttributes.All() {
			if existingAttr.Key() != sourceAttr.Key() || !existingAttr.Value().Equal(sourceAttr.Value()) {
				continue
			}
			idx, err := safeInt32(j)
			if err != nil {
				return fmt.Errorf("attribute table index: %w", err)
			}
			newIndex = idx
			found = true
			break
		}

		if !found {
			sourceAttr.CopyTo(destAttributes.AppendEmpty())
			length := destAttributes.Len() - 1
			idx, err := safeInt32(length)
			if err != nil {
				return fmt.Errorf("attribute table length: %w", err)
			}
			newIndex = idx
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("attribute index: %w", err)
		}
		attributeMapping[idx] = newIndex
	}
	return nil
}

// mergeResourceProfiles merges resource profiles with updated dictionary references
func (ms Profiles) mergeResourceProfiles(sourceResourceProfiles ResourceProfilesSlice, mappings *indexMappings) error {
	destResourceProfiles := ms.ResourceProfiles()

	for i := range sourceResourceProfiles.Len() {
		sourceRP := sourceResourceProfiles.At(i)
		destRP := destResourceProfiles.AppendEmpty()

		sourceRP.Resource().CopyTo(destRP.Resource())
		destRP.SetSchemaUrl(sourceRP.SchemaUrl())

		ms.mergeScopeProfiles(sourceRP.ScopeProfiles(), destRP.ScopeProfiles(), mappings)
	}

	return nil
}

// mergeScopeProfiles merges scope profiles with updated dictionary references
func (ms Profiles) mergeScopeProfiles(sourceScopeProfiles, destScopeProfiles ScopeProfilesSlice, mappings *indexMappings) {
	for i := range sourceScopeProfiles.Len() {
		sourceSP := sourceScopeProfiles.At(i)
		destSP := destScopeProfiles.AppendEmpty()

		sourceSP.Scope().CopyTo(destSP.Scope())
		destSP.SetSchemaUrl(sourceSP.SchemaUrl())

		ms.mergeProfiles(sourceSP.Profiles(), destSP.Profiles(), mappings)
	}
}

// mergeProfiles merges individual profile records with updated dictionary references
func (ms Profiles) mergeProfiles(sourceProfiles, destProfiles ProfilesSlice, mappings *indexMappings) {
	for i := range sourceProfiles.Len() {
		sourceProfile := sourceProfiles.At(i)
		destProfile := destProfiles.AppendEmpty()

		destProfile.SetProfileID(sourceProfile.ProfileID())
		destProfile.SetTime(sourceProfile.Time())
		destProfile.SetDuration(sourceProfile.Duration())
		sourceProfile.PeriodType().CopyTo(destProfile.PeriodType())
		destProfile.SetPeriod(sourceProfile.Period())
		destProfile.SetDefaultSampleTypeIndex(sourceProfile.DefaultSampleTypeIndex())
		destProfile.SetDroppedAttributesCount(sourceProfile.DroppedAttributesCount())
		destProfile.SetOriginalPayloadFormat(sourceProfile.OriginalPayloadFormat())
		sourceProfile.OriginalPayload().CopyTo(destProfile.OriginalPayload())

		sourceLocationIndices := sourceProfile.LocationIndices()
		destLocationIndices := destProfile.LocationIndices()
		destLocationIndices.EnsureCapacity(sourceLocationIndices.Len())
		for j := range sourceLocationIndices.Len() {
			sourceIdx := sourceLocationIndices.At(j)
			if newIdx, exists := mappings.locationTable[sourceIdx]; exists {
				destLocationIndices.Append(newIdx)
			} else {
				destLocationIndices.Append(sourceIdx)
			}
		}

		sourceAttrIndices := sourceProfile.AttributeIndices()
		destAttrIndices := destProfile.AttributeIndices()
		destAttrIndices.EnsureCapacity(sourceAttrIndices.Len())
		for j := range sourceAttrIndices.Len() {
			sourceIdx := sourceAttrIndices.At(j)
			if newIdx, exists := mappings.attributeTable[sourceIdx]; exists {
				destAttrIndices.Append(newIdx)
			} else {
				destAttrIndices.Append(sourceIdx)
			}
		}

		sourceProfile.SampleType().CopyTo(destProfile.SampleType())

		ms.mergeSamples(sourceProfile.Sample(), destProfile.Sample(), mappings)

		sourceProfile.CommentStrindices().CopyTo(destProfile.CommentStrindices())
	}
}

// mergeSamples merges samples with updated dictionary references
func (ms Profiles) mergeSamples(sourceSamples, destSamples SampleSlice, mappings *indexMappings) {
	for i := range sourceSamples.Len() {
		sourceSample := sourceSamples.At(i)
		destSample := destSamples.AppendEmpty()

		destSample.SetLocationsStartIndex(sourceSample.LocationsStartIndex())
		destSample.SetLocationsLength(sourceSample.LocationsLength())

		sourceSample.Value().CopyTo(destSample.Value())
		sourceSample.TimestampsUnixNano().CopyTo(destSample.TimestampsUnixNano())

		if linkIdx := sourceSample.LinkIndex(); linkIdx >= 0 {
			if newIdx, exists := mappings.linkTable[linkIdx]; exists {
				destSample.SetLinkIndex(newIdx)
			} else {
				destSample.SetLinkIndex(linkIdx)
			}
		} else {
			destSample.SetLinkIndex(sourceSample.LinkIndex())
		}

		sourceAttrIndices := sourceSample.AttributeIndices()
		destAttrIndices := destSample.AttributeIndices()
		destAttrIndices.EnsureCapacity(sourceAttrIndices.Len())
		for j := range sourceAttrIndices.Len() {
			sourceIdx := sourceAttrIndices.At(j)
			if newIdx, exists := mappings.attributeTable[sourceIdx]; exists {
				destAttrIndices.Append(newIdx)
			} else {
				destAttrIndices.Append(sourceIdx)
			}
		}
	}
}

// mergeAttributeUnits merges attribute units tables
func (ms Profiles) mergeAttributeUnits(destDict, sourceDict ProfilesDictionary, stringMapping map[int32]int32) {
	destAttributeUnits := destDict.AttributeUnits()
	sourceAttributeUnits := sourceDict.AttributeUnits()

	for i := range sourceAttributeUnits.Len() {
		sourceAttrUnit := sourceAttributeUnits.At(i)

		updatedAttrUnit := NewAttributeUnit()
		sourceAttrUnit.CopyTo(updatedAttrUnit)

		if keyIdx := updatedAttrUnit.AttributeKeyStrindex(); keyIdx >= 0 {
			if newIdx, exists := stringMapping[keyIdx]; exists {
				updatedAttrUnit.SetAttributeKeyStrindex(newIdx)
			}
		}
		if unitIdx := updatedAttrUnit.UnitStrindex(); unitIdx >= 0 {
			if newIdx, exists := stringMapping[unitIdx]; exists {
				updatedAttrUnit.SetUnitStrindex(newIdx)
			}
		}

		found := false
		for _, existingAttrUnit := range destAttributeUnits.All() {
			if existingAttrUnit.AttributeKeyStrindex() == updatedAttrUnit.AttributeKeyStrindex() &&
				existingAttrUnit.UnitStrindex() == updatedAttrUnit.UnitStrindex() {
				found = true
				break
			}
		}

		if !found {
			updatedAttrUnit.CopyTo(destAttributeUnits.AppendEmpty())
		}
	}
}
