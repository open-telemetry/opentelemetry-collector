// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntity_Type(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	assert.Equal(t, "service", e.Type())
}

func TestEntity_SchemaURL(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	assert.Empty(t, e.SchemaURL())

	e.SetSchemaURL("https://opentelemetry.io/schemas/1.0.0")
	assert.Equal(t, "https://opentelemetry.io/schemas/1.0.0", e.SchemaURL())

	e.SetSchemaURL("https://opentelemetry.io/schemas/1.1.0")
	assert.Equal(t, "https://opentelemetry.io/schemas/1.1.0", e.SchemaURL())
}

func TestEntity_IDAttributes(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	idAttrs := e.IDAttributes()
	idAttrs.PutStr("key1", "value1")

	val, ok := e.IDAttributes().Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())
}

func TestEntity_DescriptionAttributes(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	descAttrs := e.DescriptionAttributes()
	descAttrs.PutStr("key1", "value1")

	val, ok := e.DescriptionAttributes().Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())
}

func TestEntity_IdAndDescriptionAttributes_Isolated(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	e.IDAttributes().PutStr("id.key", "id-value")
	e.DescriptionAttributes().PutStr("desc.key", "desc-value")

	val, ok := e.IDAttributes().Get("id.key")
	assert.True(t, ok)
	assert.Equal(t, "id-value", val.Str())

	_, ok = e.IDAttributes().Get("desc.key")
	assert.False(t, ok)

	val, ok = e.DescriptionAttributes().Get("desc.key")
	assert.True(t, ok)
	assert.Equal(t, "desc-value", val.Str())

	_, ok = e.DescriptionAttributes().Get("id.key")
	assert.False(t, ok)
}

func TestEntity_IdAndDescriptionAttributes_CanPut(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	e.IDAttributes().PutStr("shared.key", "id-value")

	assert.True(t, e.IDAttributes().CanPut("shared.key"))
	assert.False(t, e.DescriptionAttributes().CanPut("shared.key"))

	e.DescriptionAttributes().PutStr("shared.key", "desc-value")

	val, ok := e.IDAttributes().Get("shared.key")
	assert.True(t, ok)
	assert.Equal(t, "desc-value", val.Str())

	val, ok = e.DescriptionAttributes().Get("shared.key")
	assert.True(t, ok)
	assert.Equal(t, "desc-value", val.Str())
}
