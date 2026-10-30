// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// MarkReadOnly marks the ResourceProfiles as shared so that no further modifications can be done on it.
func (ms Profiles) MarkReadOnly() {
	ms.getState().MarkReadOnly()
}

// IsReadOnly returns true if this ResourceProfiles instance is read-only.
func (ms Profiles) IsReadOnly() bool {
	return ms.getState().IsReadOnly()
}

// SampleCount calculates the total number of samples.
func (ms Profiles) SampleCount() int {
	sampleCount := 0
	rps := ms.ResourceProfiles()
	for i := 0; i < rps.Len(); i++ {
		rp := rps.At(i)
		sps := rp.ScopeProfiles()
		for j := 0; j < sps.Len(); j++ {
			pcs := sps.At(j).Profiles()
			for k := 0; k < pcs.Len(); k++ {
				sampleCount += pcs.At(k).Samples().Len()
			}
		}
	}
	return sampleCount
}

// switchDictionary updates the Profiles, switching its indices from one
// dictionary to another.
func (ms Profiles) switchDictionary(src, dst ProfilesDictionary) error {
	if err := validateDictionary(src); err != nil {
		return err
	}
	mi := newMergeIndex(dst)

	for i, v := range ms.ResourceProfiles().All() {
		if err := v.switchDictionary(src, dst, mi); err != nil {
			return fmt.Errorf("error switching dictionary for resource profile %d: %w", i, err)
		}
	}
	return nil
}

// ProfileCount calculates the total number of profile records.
func (ms Profiles) ProfileCount() int {
	profileCount := 0
	rps := ms.ResourceProfiles()
	for i := 0; i < rps.Len(); i++ {
		rp := rps.At(i)
		sps := rp.ScopeProfiles()
		for j := 0; j < sps.Len(); j++ {
			profileCount += sps.At(j).Profiles().Len()
		}
	}
	return profileCount
}

// validateDictionary checks every index stored in the dictionary entries,
// including entries no profile references. Only referenced entries are copied
// by switchDictionary, so this keeps MergeTo rejecting the same invalid
// dictionaries as before without copying anything.
func validateDictionary(d ProfilesDictionary) error {
	strs, attrs := d.StringTable().Len(), d.AttributeTable().Len()
	checkStr := func(idx int32, name string) error {
		if idx > 0 && strs <= int(idx) {
			return fmt.Errorf("invalid %s index %d", name, idx)
		}
		return nil
	}
	checkAttrs := func(indices pcommon.Int32Slice) error {
		for _, v := range indices.All() {
			if v < 0 || attrs <= int(v) {
				return fmt.Errorf("invalid attribute index %d", v)
			}
		}
		return nil
	}

	for i, a := range d.AttributeTable().All() {
		if err := errors.Join(checkStr(a.KeyStrindex(), "key"), checkStr(a.UnitStrindex(), "unit")); err != nil {
			return fmt.Errorf("couldn't switch attribute %d: %w", i, err)
		}
	}
	for i, fn := range d.FunctionTable().All() {
		if err := errors.Join(checkStr(fn.NameStrindex(), "name"), checkStr(fn.SystemNameStrindex(), "system name"), checkStr(fn.FilenameStrindex(), "filename")); err != nil {
			return fmt.Errorf("couldn't switch function %d: %w", i, err)
		}
	}
	for i, ma := range d.MappingTable().All() {
		if err := errors.Join(checkStr(ma.FilenameStrindex(), "filename"), checkAttrs(ma.AttributeIndices())); err != nil {
			return fmt.Errorf("couldn't switch mapping %d: %w", i, err)
		}
	}
	for i, loc := range d.LocationTable().All() {
		var err error
		if idx := loc.MappingIndex(); idx > 0 && d.MappingTable().Len() <= int(idx) {
			err = fmt.Errorf("invalid mapping index %d", idx)
		}
		err = errors.Join(err, checkAttrs(loc.AttributeIndices()))
		for j, l := range loc.Lines().All() {
			if idx := l.FunctionIndex(); idx > 0 && d.FunctionTable().Len() <= int(idx) {
				err = errors.Join(err, fmt.Errorf("couldn't switch dictionary for line %d: invalid function index %d", j, idx))
			}
		}
		if err != nil {
			return fmt.Errorf("couldn't switch location %d: %w", i, err)
		}
	}
	for i, st := range d.StackTable().All() {
		for _, v := range st.LocationIndices().All() {
			if v < 0 || d.LocationTable().Len() <= int(v) {
				return fmt.Errorf("couldn't switch stack %d: invalid location index %d", i, v)
			}
		}
	}
	return nil
}
