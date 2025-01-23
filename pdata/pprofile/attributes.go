// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type attributable interface {
	AttributeIndices() pcommon.Int32Slice
}

// BuildAttributes builds a [pcommon.Map] containing the attributes of a record.
// The record can by any struct that implements an `AttributeIndices` method.
func BuildAttributes(profile Profile, record attributable) (pcommon.Map, error) {
	attrs := map[string]any{}

	for i := range record.AttributeIndices().AsRaw() {
		a := profile.AttributeTable().At(i)
		attrs[a.Key()] = a.Value().AsRaw()
	}

	m := pcommon.NewMap()
	err := m.FromRaw(attrs)
	return m, err
}
