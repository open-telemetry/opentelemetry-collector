// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type attributable interface {
	AttributeIndices() pcommon.Int32Slice
}

// FromAttributeIndices builds a [pcommon.Map] containing the attributes of a
// record.
// The record can by any struct that implements an `AttributeIndices` method.
func FromAttributeIndices(profile Profile, record attributable) pcommon.Map {
	m := pcommon.NewMap()
	m.EnsureCapacity(record.AttributeIndices().Len())

	for i := 0; i < record.AttributeIndices().Len(); i++ {
		kv := profile.AttributeTable().At(int(record.AttributeIndices().At(i)))
		kv.Value().CopyTo(m.PutEmpty(kv.Key()))
	}

	return m
}
