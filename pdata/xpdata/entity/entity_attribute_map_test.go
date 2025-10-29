// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestEntityAttributeMap(t *testing.T) {
	m := newTestEntityAttributeMap()

	val, ok := m.Get("non-existent")
	assert.False(t, ok)
	assert.Equal(t, pcommon.Value{}, val)

	m.PutStr("k1", "v1")
	val, ok = m.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, "v1", val.Str())

	assert.True(t, m.Remove("k1"))
	_, ok = m.Get("k1")
	assert.False(t, ok)

	assert.False(t, m.Remove("non-existent"))
}

func TestEntityAttributeMap_Get(t *testing.T) {
	m := newTestEntityAttributeMap()
	m.attributes.PutStr("owned", "value1")
	m.attributes.PutStr("not-owned", "value2")
	m.keys.Append("owned")

	val, ok := m.Get("owned")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())

	_, ok = m.Get("not-owned")
	assert.False(t, ok)

	_, ok = m.Get("non-existent")
	assert.False(t, ok)
}

func TestEntityAttributeMap_PutStr(t *testing.T) {
	m := newTestEntityAttributeMap()

	m.PutStr("k1", "v1")
	assert.Equal(t, 1, m.keys.Len())
	assert.Equal(t, "k1", m.keys.At(0))
	val, ok := m.attributes.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, "v1", val.Str())

	m.PutStr("k1", "v2")
	assert.Equal(t, 1, m.keys.Len())
	val, ok = m.attributes.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, "v2", val.Str())
}

func TestEntityAttributeMap_PutStr_Overwrites(t *testing.T) {
	m := newTestEntityAttributeMap()
	m.attributes.PutStr("existing", "old-value")

	m.PutStr("existing", "new-value")

	assert.Equal(t, 1, m.keys.Len())
	assert.Equal(t, "existing", m.keys.At(0))
	val, ok := m.attributes.Get("existing")
	assert.True(t, ok)
	assert.Equal(t, "new-value", val.Str())
}

func TestEntityAttributeMap_PutEmpty(t *testing.T) {
	m := newTestEntityAttributeMap()

	val := m.PutEmpty("k1")
	val.SetStr("v1")

	assert.Equal(t, 1, m.keys.Len())
	assert.Equal(t, "k1", m.keys.At(0))
	resVal, ok := m.attributes.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, "v1", resVal.Str())
}

func TestEntityAttributeMap_PutEmpty_Overwrites(t *testing.T) {
	m := newTestEntityAttributeMap()
	m.attributes.PutStr("existing", "old-value")

	val := m.PutEmpty("existing")
	val.SetStr("new-value")

	assert.Equal(t, 1, m.keys.Len())
	assert.Equal(t, "existing", m.keys.At(0))
	resVal, ok := m.attributes.Get("existing")
	assert.True(t, ok)
	assert.Equal(t, "new-value", resVal.Str())
}

func TestEntityAttributeMap_Remove(t *testing.T) {
	m := newTestEntityAttributeMap()
	m.keys.Append("k1")
	m.keys.Append("k2")
	m.attributes.PutStr("k1", "v1")
	m.attributes.PutStr("k2", "v2")

	assert.True(t, m.Remove("k1"))
	assert.Equal(t, 1, m.keys.Len())
	assert.Equal(t, "k2", m.keys.At(0))
	_, ok := m.attributes.Get("k1")
	assert.False(t, ok)

	val, ok := m.attributes.Get("k2")
	assert.True(t, ok)
	assert.Equal(t, "v2", val.Str())
}

func TestEntityAttributeMap_Remove_NotOwned(t *testing.T) {
	m := newTestEntityAttributeMap()
	m.attributes.PutStr("not-owned", "value")

	assert.False(t, m.Remove("not-owned"))

	val, ok := m.attributes.Get("not-owned")
	assert.True(t, ok)
	assert.Equal(t, "value", val.Str())
}

func TestEntityAttributeMap_Remove_NonExistent(t *testing.T) {
	m := newTestEntityAttributeMap()

	assert.False(t, m.Remove("non-existent"))
}

func TestEntityAttributeMap_CanPut(t *testing.T) {
	m := newTestEntityAttributeMap()
	m.keys.Append("owned")
	m.attributes.PutStr("owned", "value")
	m.attributes.PutStr("other", "value")

	assert.True(t, m.CanPut("owned"))
	assert.False(t, m.CanPut("other"))
	assert.True(t, m.CanPut("new-key"))
}

func newTestEntityAttributeMap() EntityAttributeMap {
	return EntityAttributeMap{
		keys:       pcommon.NewStringSlice(),
		attributes: pcommon.NewMap(),
	}
}
