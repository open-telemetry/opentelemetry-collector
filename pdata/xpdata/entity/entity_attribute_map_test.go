// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestEntityAttributeMap_PutStr(t *testing.T) {
	attributes := pcommon.NewMap()
	keys := pcommon.NewStringSlice()
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	err := m.PutStr("key1", "value1")
	require.NoError(t, err)

	val, ok := attributes.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())
	assert.Equal(t, 1, keys.Len())
	assert.Equal(t, "key1", keys.At(0))
}

func TestEntityAttributeMap_PutStr_Update(t *testing.T) {
	attributes := pcommon.NewMap()
	keys := pcommon.NewStringSlice()
	keys.Append("key1")
	attributes.PutStr("key1", "value1")
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	err := m.PutStr("key1", "value2")
	require.NoError(t, err)

	val, ok := attributes.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value2", val.Str())
	assert.Equal(t, 1, keys.Len())
}

func TestEntityAttributeMap_PutStr_Conflict(t *testing.T) {
	attributes := pcommon.NewMap()
	attributes.PutStr("conflicting", "existing-value")
	keys := pcommon.NewStringSlice()
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	err := m.PutStr("conflicting", "new-value")
	require.ErrorIs(t, err, ErrConflictingAttribute)

	val, ok := attributes.Get("conflicting")
	assert.True(t, ok)
	assert.Equal(t, "existing-value", val.Str())
	assert.Equal(t, 0, keys.Len())
}

func TestEntityAttributeMap_PutEmpty(t *testing.T) {
	attributes := pcommon.NewMap()
	keys := pcommon.NewStringSlice()
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	val, err := m.PutEmpty("key1")
	require.NoError(t, err)
	val.SetStr("value1")

	resVal, ok := attributes.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", resVal.Str())
	assert.Equal(t, 1, keys.Len())
	assert.Equal(t, "key1", keys.At(0))
}

func TestEntityAttributeMap_PutEmpty_Conflict(t *testing.T) {
	attributes := pcommon.NewMap()
	attributes.PutStr("conflicting", "existing-value")
	keys := pcommon.NewStringSlice()
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	_, err := m.PutEmpty("conflicting")
	require.ErrorIs(t, err, ErrConflictingAttribute)

	val, ok := attributes.Get("conflicting")
	assert.True(t, ok)
	assert.Equal(t, "existing-value", val.Str())
	assert.Equal(t, 0, keys.Len())
}

func TestEntityAttributeMap_Get(t *testing.T) {
	attributes := pcommon.NewMap()
	attributes.PutStr("key1", "value1")
	attributes.PutStr("key2", "value2")
	keys := pcommon.NewStringSlice()
	keys.Append("key1")
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	val, ok := m.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())

	_, ok = m.Get("key2")
	assert.False(t, ok)

	_, ok = m.Get("non-existent")
	assert.False(t, ok)
}

func TestEntityAttributeMap_Remove(t *testing.T) {
	attributes := pcommon.NewMap()
	attributes.PutStr("key1", "value1")
	attributes.PutStr("key2", "value2")
	keys := pcommon.NewStringSlice()
	keys.Append("key1")
	keys.Append("key2")
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	removed := m.Remove("key1")
	assert.True(t, removed)

	_, ok := attributes.Get("key1")
	assert.False(t, ok)
	assert.Equal(t, 1, keys.Len())
}

func TestEntityAttributeMap_Remove_NotInKeys(t *testing.T) {
	attributes := pcommon.NewMap()
	attributes.PutStr("key1", "value1")
	keys := pcommon.NewStringSlice()
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	removed := m.Remove("key1")
	assert.False(t, removed)

	val, ok := attributes.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())
}

func TestEntityAttributeMap_Remove_NonExistent(t *testing.T) {
	attributes := pcommon.NewMap()
	keys := pcommon.NewStringSlice()
	m := EntityAttributeMap{keys: keys, attributes: attributes}

	removed := m.Remove("non-existent")
	assert.False(t, removed)
}
