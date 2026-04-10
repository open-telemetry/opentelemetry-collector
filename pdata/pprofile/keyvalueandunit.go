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
func (ms KeyValueAndUnit) switchDictionary(src, dst ProfilesDictionary) error {
	if src.StringTable().Len() <= int(ms.KeyStrindex()) {
		return fmt.Errorf("invalid key index %d", ms.KeyStrindex())
	}

	keyIdx, err := SetString(dst.StringTable(), src.StringTable().At(int(ms.KeyStrindex())))
	if err != nil {
		return fmt.Errorf("couldn't set key: %w", err)
	}
	ms.SetKeyStrindex(keyIdx)

	if src.StringTable().Len() <= int(ms.UnitStrindex()) {
		return fmt.Errorf("invalid unit index %d", ms.UnitStrindex())
	}

	unitIdx, err := SetString(dst.StringTable(), src.StringTable().At(int(ms.UnitStrindex())))
	if err != nil {
		return fmt.Errorf("couldn't set unit: %w", err)
	}
	ms.SetUnitStrindex(unitIdx)

	return nil
}
