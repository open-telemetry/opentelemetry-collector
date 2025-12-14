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

func TestEntity_IdentifyingAttributes(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	idAttrs := e.IdentifyingAttributes()
	idAttrs.PutStr("key1", "value1")

	val, ok := e.IdentifyingAttributes().Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())
}

func TestEntity_DescriptiveAttributes(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	descAttrs := e.DescriptiveAttributes()
	descAttrs.PutStr("key1", "value1")

	val, ok := e.DescriptiveAttributes().Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val.Str())
}

func TestEntity_IdAndDescriptionAttributes_Isolated(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	e.IdentifyingAttributes().PutStr("id.key", "id-value")
	e.DescriptiveAttributes().PutStr("desc.key", "desc-value")

	val, ok := e.IdentifyingAttributes().Get("id.key")
	assert.True(t, ok)
	assert.Equal(t, "id-value", val.Str())

	_, ok = e.IdentifyingAttributes().Get("desc.key")
	assert.False(t, ok)

	val, ok = e.DescriptiveAttributes().Get("desc.key")
	assert.True(t, ok)
	assert.Equal(t, "desc-value", val.Str())

	_, ok = e.DescriptiveAttributes().Get("id.key")
	assert.False(t, ok)
}

func TestEntity_IdAndDescriptionAttributes_CanPut(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")

	e.IdentifyingAttributes().PutStr("shared.key", "id-value")

	assert.True(t, e.IdentifyingAttributes().CanPut("shared.key"))
	assert.False(t, e.DescriptiveAttributes().CanPut("shared.key"))

	e.DescriptiveAttributes().PutStr("shared.key", "desc-value")

	val, ok := e.IdentifyingAttributes().Get("shared.key")
	assert.True(t, ok)
	assert.Equal(t, "desc-value", val.Str())

	val, ok = e.DescriptiveAttributes().Get("shared.key")
	assert.True(t, ok)
	assert.Equal(t, "desc-value", val.Str())
}
