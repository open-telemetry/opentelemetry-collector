// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xhash

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestMapHash_SameContentDifferentOrder(t *testing.T) {
	m1 := pcommon.NewMap()
	m1.PutStr("a", "1")
	m1.PutStr("b", "2")

	m2 := pcommon.NewMap()
	m2.PutStr("b", "2")
	m2.PutStr("a", "1")

	assert.Equal(t, MapHash(m1), MapHash(m2))
}

func TestMapHash_DifferentContent(t *testing.T) {
	m1 := pcommon.NewMap()
	m1.PutStr("a", "1")

	m2 := pcommon.NewMap()
	m2.PutStr("a", "2")

	assert.NotEqual(t, MapHash(m1), MapHash(m2))
}

func TestMapHash_TypeMatters(t *testing.T) {
	m1 := pcommon.NewMap()
	m1.PutStr("a", "1")

	m2 := pcommon.NewMap()
	m2.PutInt("a", 1)

	assert.NotEqual(t, MapHash(m1), MapHash(m2))
}

func TestMapHash_Empty(t *testing.T) {
	m := pcommon.NewMap()
	assert.Equal(t, MapHash(m), MapHash(pcommon.NewMap()))
}

func TestMapHash_Nested(t *testing.T) {
	m1 := pcommon.NewMap()
	nested1 := m1.PutEmptyMap("nested")
	nested1.PutStr("x", "y")

	m2 := pcommon.NewMap()
	nested2 := m2.PutEmptyMap("nested")
	nested2.PutStr("x", "y")

	assert.Equal(t, MapHash(m1), MapHash(m2))
}

func TestMapHash_Slice(t *testing.T) {
	m1 := pcommon.NewMap()
	s1 := m1.PutEmptySlice("items")
	s1.AppendEmpty().SetStr("a")
	s1.AppendEmpty().SetStr("b")

	m2 := pcommon.NewMap()
	s2 := m2.PutEmptySlice("items")
	s2.AppendEmpty().SetStr("a")
	s2.AppendEmpty().SetStr("b")

	assert.Equal(t, MapHash(m1), MapHash(m2))
}

// A zero-value key must not hash the same as a missing key.
func TestMapHash_ZeroValueVsAbsentKey(t *testing.T) {
	absent := pcommon.NewMap()

	str := pcommon.NewMap()
	str.PutStr("a", "")
	assert.NotEqual(t, MapHash(absent), MapHash(str))

	i := pcommon.NewMap()
	i.PutInt("a", 0)
	assert.NotEqual(t, MapHash(absent), MapHash(i))

	b := pcommon.NewMap()
	b.PutBool("a", false)
	assert.NotEqual(t, MapHash(absent), MapHash(b))

	sl := pcommon.NewMap()
	sl.PutEmptySlice("a")
	assert.NotEqual(t, MapHash(absent), MapHash(sl))
}

// Key/value byte boundaries must not be ambiguous.
func TestMapHash_KeyValueBoundary(t *testing.T) {
	m1 := pcommon.NewMap()
	m1.PutStr("ab", "c")

	m2 := pcommon.NewMap()
	m2.PutStr("a", "bc")

	assert.NotEqual(t, MapHash(m1), MapHash(m2))
}

// A nested map must not collide with a flat key that looks like the path.
func TestMapHash_NestedVsFlatKey(t *testing.T) {
	m1 := pcommon.NewMap()
	nested := m1.PutEmptyMap("a")
	nested.PutStr("b", "c")

	m2 := pcommon.NewMap()
	m2.PutStr("a.b", "c")

	assert.NotEqual(t, MapHash(m1), MapHash(m2))
}

// Bytes and a string with the same content must hash differently.
func TestMapHash_BytesVsStr(t *testing.T) {
	m1 := pcommon.NewMap()
	m1.PutStr("a", "hi")

	m2 := pcommon.NewMap()
	m2.PutEmptyBytes("a").FromRaw([]byte("hi"))

	assert.NotEqual(t, MapHash(m1), MapHash(m2))
}
