// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// Equal checks equality with another KeyValueAndUnit
// It assumes both structs refer to the same dictionary.
func (ms KeyValueAndUnit) Equal(val KeyValueAndUnit) bool {
	return ms.KeyStrindex() == val.KeyStrindex() &&
		ms.UnitStrindex() == val.UnitStrindex() &&
		ms.Value().Equal(val.Value())
}

// switchDictionary updates the KeyValueAndUnit, switching its indices from one
// dictionary to another.
func (ms KeyValueAndUnit) switchDictionary(src, dst ProfilesDictionary, mi *mergeIndex) error {
	key, unit, err := ms.translateStrings(src, dst, mi)
	if err != nil {
		return err
	}
	ms.SetKeyStrindex(key)
	ms.SetUnitStrindex(unit)
	return nil
}

// translateStrings returns the destination key and unit string indices of the
// KeyValueAndUnit, adding the strings to dst as needed, without modifying it.
func (ms KeyValueAndUnit) translateStrings(src, dst ProfilesDictionary, mi *mergeIndex) (key, unit int32, err error) {
	key, unit = ms.KeyStrindex(), ms.UnitStrindex()

	if key > 0 {
		if src.StringTable().Len() <= int(key) {
			return 0, 0, fmt.Errorf("invalid key index %d", key)
		}

		key, err = mi.setString(dst.StringTable(), src.StringTable().At(int(key)))
		if err != nil {
			return 0, 0, fmt.Errorf("couldn't set key: %w", err)
		}
	}

	if unit > 0 {
		if src.StringTable().Len() <= int(unit) {
			return 0, 0, fmt.Errorf("invalid unit index %d", unit)
		}

		unit, err = mi.setString(dst.StringTable(), src.StringTable().At(int(unit)))
		if err != nil {
			return 0, 0, fmt.Errorf("couldn't set unit: %w", err)
		}
	}

	return key, unit, nil
}
