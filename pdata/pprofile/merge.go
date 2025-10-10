// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func safeInt32(i int) (int32, error) {
	if i < 0 || i > math.MaxInt32 {
		return 0, fmt.Errorf("value %d outside int32 range", i)
	}
	return int32(i), nil
}

var (
	ErrDictionarySizeExceeded = fmt.Errorf("dictionary size would exceed maximum limit of %d entries", math.MaxInt32)
	ErrSourceReadOnly         = errors.New("cannot merge from read-only source")
	ErrDestinationReadOnly    = errors.New("cannot merge into read-only destination")
)

// MergeFrom merges the profiles from the source Profiles into the current Profiles instance.
// This operation deduplicates dictionary entries and updates all references accordingly.
func (ms Profiles) MergeFrom(source Profiles) error {
	if ms.IsReadOnly() {
		return ErrDestinationReadOnly
	}
	if source.IsReadOnly() {
		return ErrSourceReadOnly
	}

	destDict := ms.Dictionary()
	sourceDict := source.Dictionary()

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
	stackTable     map[int32]int32
	linkTable      map[int32]int32
	attributeTable map[int32]int32
}

func (ms Profiles) mergeDictionaries(destDict, sourceDict ProfilesDictionary) (*indexMappings, error) {
	mappings := &indexMappings{
		stringTable:    make(map[int32]int32),
		functionTable:  make(map[int32]int32),
		mappingTable:   make(map[int32]int32),
		locationTable:  make(map[int32]int32),
		stackTable:     make(map[int32]int32),
		linkTable:      make(map[int32]int32),
		attributeTable: make(map[int32]int32),
	}

	if err := ms.checkSizeLimits(destDict, sourceDict); err != nil {
		return nil, err
	}

	if err := ms.mergeStringTable(destDict, sourceDict, mappings.stringTable); err != nil {
		return nil, err
	}

	if err := ms.mergeAttributeTable(destDict, sourceDict, mappings.attributeTable, mappings.stringTable); err != nil {
		return nil, err
	}

	if err := ms.mergeFunctionTable(destDict, sourceDict, mappings.functionTable, mappings.stringTable); err != nil {
		return nil, err
	}

	if err := ms.mergeMappingTable(destDict, sourceDict, mappings.mappingTable, mappings.stringTable); err != nil {
		return nil, err
	}

	if err := ms.mergeLocationTable(destDict, sourceDict, mappings.locationTable, mappings.mappingTable, mappings.functionTable); err != nil {
		return nil, err
	}

	if err := ms.mergeStackTable(destDict, sourceDict, mappings.stackTable, mappings.locationTable); err != nil {
		return nil, err
	}

	if err := ms.mergeLinkTable(destDict, sourceDict, mappings.linkTable); err != nil {
		return nil, err
	}

	return mappings, nil
}

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
	if int64(destDict.StackTable().Len())+int64(sourceDict.StackTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.LinkTable().Len())+int64(sourceDict.LinkTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	if int64(destDict.AttributeTable().Len())+int64(sourceDict.AttributeTable().Len()) > math.MaxInt32 {
		return ErrDictionarySizeExceeded
	}
	return nil
}

func (ms Profiles) mergeStringTable(destDict, sourceDict ProfilesDictionary, stringMapping map[int32]int32) error {
	destStrings := destDict.StringTable()
	sourceStrings := sourceDict.StringTable()

	for i := 0; i < sourceStrings.Len(); i++ {
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

func (ms Profiles) mergeFunctionTable(destDict, sourceDict ProfilesDictionary, functionMapping, stringMapping map[int32]int32) error {
	destFunctions := destDict.FunctionTable()
	sourceFunctions := sourceDict.FunctionTable()

	for i := 0; i < sourceFunctions.Len(); i++ {
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

		line := NewLine()
		if err := SetFunction(destFunctions, line, updatedFunc); err != nil {
			return fmt.Errorf("failed to merge function table entry at index %d: %w", i, err)
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("function mapping index: %w", err)
		}
		functionMapping[idx] = line.FunctionIndex()
	}
	return nil
}

func (ms Profiles) mergeMappingTable(destDict, sourceDict ProfilesDictionary, mappingMapping, stringMapping map[int32]int32) error {
	destMappings := destDict.MappingTable()
	sourceMappings := sourceDict.MappingTable()

	for i := 0; i < sourceMappings.Len(); i++ {
		sourceMapping := sourceMappings.At(i)

		updatedMapping := NewMapping()
		sourceMapping.CopyTo(updatedMapping)

		if filenameIdx := updatedMapping.FilenameStrindex(); filenameIdx >= 0 {
			if newIdx, exists := stringMapping[filenameIdx]; exists {
				updatedMapping.SetFilenameStrindex(newIdx)
			}
		}

		location := NewLocation()
		if err := SetMapping(destMappings, location, updatedMapping); err != nil {
			return fmt.Errorf("failed to merge mapping table entry at index %d: %w", i, err)
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("mapping index: %w", err)
		}
		mappingMapping[idx] = location.MappingIndex()
	}
	return nil
}

func (ms Profiles) mergeLocationTable(destDict, sourceDict ProfilesDictionary, locationMapping, mappingMapping, functionMapping map[int32]int32) error {
	destLocations := destDict.LocationTable()
	sourceLocations := sourceDict.LocationTable()

	for i := 0; i < sourceLocations.Len(); i++ {
		sourceLocation := sourceLocations.At(i)

		updatedLocation := NewLocation()
		sourceLocation.CopyTo(updatedLocation)

		if mappingIdx := updatedLocation.MappingIndex(); mappingIdx >= 0 {
			if newIdx, exists := mappingMapping[mappingIdx]; exists {
				updatedLocation.SetMappingIndex(newIdx)
			}
		}

		lines := updatedLocation.Line()
		for j := 0; j < lines.Len(); j++ {
			line := lines.At(j)
			if funcIdx := line.FunctionIndex(); funcIdx >= 0 {
				if newIdx, exists := functionMapping[funcIdx]; exists {
					line.SetFunctionIndex(newIdx)
				}
			}
		}

		stack := NewStack()
		if err := PutLocation(destLocations, stack, updatedLocation); err != nil {
			return fmt.Errorf("failed to merge location table entry at index %d: %w", i, err)
		}

		if stack.LocationIndices().Len() == 0 {
			return fmt.Errorf("location indices not updated for entry %d", i)
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("location index: %w", err)
		}
		locationMapping[idx] = stack.LocationIndices().At(stack.LocationIndices().Len() - 1)
	}
	return nil
}

func (ms Profiles) mergeStackTable(destDict, sourceDict ProfilesDictionary, stackMapping, locationMapping map[int32]int32) error {
	destStacks := destDict.StackTable()
	sourceStacks := sourceDict.StackTable()

	for i := 0; i < sourceStacks.Len(); i++ {
		sourceStack := sourceStacks.At(i)

		updatedStack := NewStack()
		sourceStack.CopyTo(updatedStack)

		indices := updatedStack.LocationIndices()
		for j := 0; j < indices.Len(); j++ {
			locIdx := indices.At(j)
			if newIdx, exists := locationMapping[locIdx]; exists {
				indices.SetAt(j, newIdx)
			}
		}

		var newIndex int32
		found := false
		for j, existingStack := range destStacks.All() {
			if !stacksEqual(existingStack, updatedStack) {
				continue
			}
			idx, err := safeInt32(j)
			if err != nil {
				return fmt.Errorf("stack table index: %w", err)
			}
			newIndex = idx
			found = true
			break
		}

		if !found {
			updatedStack.CopyTo(destStacks.AppendEmpty())
			length := destStacks.Len() - 1
			idx, err := safeInt32(length)
			if err != nil {
				return fmt.Errorf("stack table length: %w", err)
			}
			newIndex = idx
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("stack mapping index: %w", err)
		}
		stackMapping[idx] = newIndex
	}
	return nil
}

func stacksEqual(a, b Stack) bool {
	aIdx := a.LocationIndices()
	bIdx := b.LocationIndices()
	if aIdx.Len() != bIdx.Len() {
		return false
	}
	for i := 0; i < aIdx.Len(); i++ {
		if aIdx.At(i) != bIdx.At(i) {
			return false
		}
	}
	return true
}

func (ms Profiles) mergeLinkTable(destDict, sourceDict ProfilesDictionary, linkMapping map[int32]int32) error {
	destLinks := destDict.LinkTable()
	sourceLinks := sourceDict.LinkTable()

	for i := 0; i < sourceLinks.Len(); i++ {
		sourceLink := sourceLinks.At(i)

		sample := NewSample()
		if err := SetLink(destLinks, sample, sourceLink); err != nil {
			return fmt.Errorf("failed to merge link table entry at index %d: %w", i, err)
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("link index: %w", err)
		}
		linkMapping[idx] = sample.LinkIndex()
	}
	return nil
}

func (ms Profiles) mergeAttributeTable(destDict, sourceDict ProfilesDictionary, attributeMapping, stringMapping map[int32]int32) error {
	destAttributes := destDict.AttributeTable()
	sourceAttributes := sourceDict.AttributeTable()

	for i := 0; i < sourceAttributes.Len(); i++ {
		sourceAttr := sourceAttributes.At(i)

		keyIdx := sourceAttr.KeyStrindex()
		if keyIdx < 0 || int(keyIdx) >= sourceDict.StringTable().Len() {
			return fmt.Errorf("attribute key index %d out of range for entry %d", keyIdx, i)
		}
		key := sourceDict.StringTable().At(int(keyIdx))
		profile := NewProfile()
		if err := PutAttribute(destAttributes, profile, destDict, key, sourceAttr.Value()); err != nil {
			return fmt.Errorf("failed to merge attribute table entry at index %d: %w", i, err)
		}

		if profile.AttributeIndices().Len() == 0 {
			return fmt.Errorf("attribute indices not updated for entry %d", i)
		}

		mappedIdx := profile.AttributeIndices().At(profile.AttributeIndices().Len() - 1)

		if unitIdx := sourceAttr.UnitStrindex(); unitIdx >= 0 {
			if newUnitIdx, exists := stringMapping[unitIdx]; exists {
				destAttributes.At(int(mappedIdx)).SetUnitStrindex(newUnitIdx)
			}
		}

		idx, err := safeInt32(i)
		if err != nil {
			return fmt.Errorf("attribute index: %w", err)
		}
		attributeMapping[idx] = mappedIdx
	}
	return nil
}

func (ms Profiles) mergeResourceProfiles(sourceResourceProfiles ResourceProfilesSlice, mappings *indexMappings) error {
	destResourceProfiles := ms.ResourceProfiles()

	for i := 0; i < sourceResourceProfiles.Len(); i++ {
		sourceRP := sourceResourceProfiles.At(i)
		destRP := destResourceProfiles.AppendEmpty()

		sourceRP.Resource().CopyTo(destRP.Resource())
		destRP.SetSchemaUrl(sourceRP.SchemaUrl())

		ms.mergeScopeProfiles(sourceRP.ScopeProfiles(), destRP.ScopeProfiles(), mappings)
	}

	return nil
}

func (ms Profiles) mergeScopeProfiles(sourceScopeProfiles, destScopeProfiles ScopeProfilesSlice, mappings *indexMappings) {
	for i := 0; i < sourceScopeProfiles.Len(); i++ {
		sourceSP := sourceScopeProfiles.At(i)
		destSP := destScopeProfiles.AppendEmpty()

		sourceSP.Scope().CopyTo(destSP.Scope())
		destSP.SetSchemaUrl(sourceSP.SchemaUrl())

		ms.mergeProfiles(sourceSP.Profiles(), destSP.Profiles(), mappings)
	}
}

func (ms Profiles) mergeProfiles(sourceProfiles, destProfiles ProfilesSlice, mappings *indexMappings) {
	for i := 0; i < sourceProfiles.Len(); i++ {
		sourceProfile := sourceProfiles.At(i)
		destProfile := destProfiles.AppendEmpty()

		destProfile.SetProfileID(sourceProfile.ProfileID())
		destProfile.SetTime(sourceProfile.Time())
		destProfile.SetDuration(sourceProfile.Duration())
		copyValueType(sourceProfile.PeriodType(), destProfile.PeriodType(), mappings.stringTable)
		destProfile.SetPeriod(sourceProfile.Period())
		destProfile.SetDroppedAttributesCount(sourceProfile.DroppedAttributesCount())
		destProfile.SetOriginalPayloadFormat(sourceProfile.OriginalPayloadFormat())
		sourceProfile.OriginalPayload().CopyTo(destProfile.OriginalPayload())

		appendMappedIndices(sourceProfile.AttributeIndices(), destProfile.AttributeIndices(), mappings.attributeTable)

		copyValueType(sourceProfile.SampleType(), destProfile.SampleType(), mappings.stringTable)

		ms.mergeSamples(sourceProfile.Sample(), destProfile.Sample(), mappings)

		appendMappedIndices(sourceProfile.CommentStrindices(), destProfile.CommentStrindices(), mappings.stringTable)
	}
}

func copyValueType(src, dest ValueType, stringMapping map[int32]int32) {
	src.CopyTo(dest)
	applyStringMappingToValueType(dest, stringMapping)
}

func applyStringMappingToValueType(v ValueType, stringMapping map[int32]int32) {
	if idx := v.TypeStrindex(); idx >= 0 {
		if newIdx, exists := stringMapping[idx]; exists {
			v.SetTypeStrindex(newIdx)
		}
	}
	if idx := v.UnitStrindex(); idx >= 0 {
		if newIdx, exists := stringMapping[idx]; exists {
			v.SetUnitStrindex(newIdx)
		}
	}
}

func appendMappedIndices(sourceIndices, destIndices pcommon.Int32Slice, mapping map[int32]int32) {
	destIndices.EnsureCapacity(sourceIndices.Len())
	for i := 0; i < sourceIndices.Len(); i++ {
		idx := sourceIndices.At(i)
		if newIdx, exists := mapping[idx]; exists {
			destIndices.Append(newIdx)
			continue
		}
		destIndices.Append(idx)
	}
}

func (ms Profiles) mergeSamples(sourceSamples, destSamples SampleSlice, mappings *indexMappings) {
	for i := 0; i < sourceSamples.Len(); i++ {
		sourceSample := sourceSamples.At(i)
		destSample := destSamples.AppendEmpty()

		stackIdx := sourceSample.StackIndex()
		if newIdx, exists := mappings.stackTable[stackIdx]; exists {
			destSample.SetStackIndex(newIdx)
		} else {
			destSample.SetStackIndex(stackIdx)
		}

		sourceSample.Values().CopyTo(destSample.Values())
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

		appendMappedIndices(sourceSample.AttributeIndices(), destSample.AttributeIndices(), mappings.attributeTable)
	}
}
