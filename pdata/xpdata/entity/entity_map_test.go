// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntityMap(t *testing.T) {
	em := NewEntityMap()
	assert.Equal(t, 0, em.Len())

	e1 := em.PutEmpty("service")
	assert.Equal(t, 1, em.Len())
	assert.Equal(t, "service", e1.Type())

	e2 := em.PutEmpty("host")
	assert.Equal(t, 2, em.Len())
	assert.Equal(t, "host", e2.Type())
}

func TestEntityMap_PutEmpty(t *testing.T) {
	em := NewEntityMap()
	e := em.PutEmpty("service")
	retrieved, ok := em.Get("service")
	assert.True(t, ok)
	assert.Equal(t, "service", retrieved.Type())
	assert.Equal(t, "service", e.Type())
}

func TestEntityMap_PutEmpty_Override(t *testing.T) {
	em := NewEntityMap()
	e1 := em.PutEmpty("service")
	e1.IdentifyingAttributes().PutStr("service.name", "my-service")
	assert.Equal(t, 1, em.Len())

	e2 := em.PutEmpty("service")
	assert.Equal(t, 1, em.Len())

	_, ok := e2.IdentifyingAttributes().Get("service.name")
	assert.False(t, ok)
}

func TestEntityMap_EnsureCapacity(t *testing.T) {
	em := NewEntityMap()
	em.EnsureCapacity(5)
	for i := range 5 {
		em.PutEmpty(fmt.Sprintf("type%d", i))
	}
	assert.Equal(t, 5, em.Len())

	em.EnsureCapacity(3)
	assert.Equal(t, 5, em.Len())
}

func TestEntityMap_AttributesIsolation(t *testing.T) {
	em := NewEntityMap()

	e1 := em.PutEmpty("service")
	e1.IdentifyingAttributes().PutStr("service.name", "my-service")

	e2 := em.PutEmpty("host")
	e2.IdentifyingAttributes().PutStr("host.name", "my-host")

	service, ok := em.Get("service")
	assert.True(t, ok)
	val, ok := service.IdentifyingAttributes().Get("service.name")
	assert.True(t, ok)
	assert.Equal(t, "my-service", val.Str())

	host, ok := em.Get("host")
	assert.True(t, ok)
	val, ok = host.IdentifyingAttributes().Get("host.name")
	assert.True(t, ok)
	assert.Equal(t, "my-host", val.Str())

	_, ok = service.IdentifyingAttributes().Get("host.name")
	assert.False(t, ok)

	_, ok = host.IdentifyingAttributes().Get("service.name")
	assert.False(t, ok)
}

func TestEntityMap_Get(t *testing.T) {
	em := NewEntityMap()
	e1 := em.PutEmpty("service")
	e1.IdentifyingAttributes().PutStr("service.name", "my-service")

	e2, ok := em.Get("service")
	assert.True(t, ok)
	assert.Equal(t, "service", e2.Type())
	val, ok := e2.IdentifyingAttributes().Get("service.name")
	assert.True(t, ok)
	assert.Equal(t, "my-service", val.Str())

	_, ok = em.Get("host")
	assert.False(t, ok)
}

func TestEntityMap_Remove(t *testing.T) {
	em := NewEntityMap()
	e1 := em.PutEmpty("service")
	e1.IdentifyingAttributes().PutStr("service.name", "my-service")
	e1.DescriptiveAttributes().PutStr("service.version", "1.0.0")

	assert.Equal(t, 1, em.Len())
	assert.Equal(t, 2, em.attributes.Len())

	removed := em.Remove("service")
	assert.True(t, removed)
	assert.Empty(t, em.Len())
	assert.Empty(t, em.attributes.Len())

	assert.False(t, em.Remove("service"))
}

func TestEntityMap_All(t *testing.T) {
	em := NewEntityMap()
	e1 := em.PutEmpty("service")
	e1.IdentifyingAttributes().PutStr("service.name", "my-service")

	e2 := em.PutEmpty("host")
	e2.IdentifyingAttributes().PutStr("host.name", "my-host")

	e3 := em.PutEmpty("process")
	e3.IdentifyingAttributes().PutStr("process.pid", "1234")

	types := make(map[string]bool)
	attributes := make(map[string]string)

	for entityType, entity := range em.All() {
		types[entityType] = true
		switch entityType {
		case "service":
			val, ok := entity.IdentifyingAttributes().Get("service.name")
			assert.True(t, ok)
			attributes[entityType] = val.Str()
		case "host":
			val, ok := entity.IdentifyingAttributes().Get("host.name")
			assert.True(t, ok)
			attributes[entityType] = val.Str()
		case "process":
			val, ok := entity.IdentifyingAttributes().Get("process.pid")
			assert.True(t, ok)
			attributes[entityType] = val.Str()
		}
	}

	assert.Len(t, types, 3)
	assert.True(t, types["service"])
	assert.True(t, types["host"])
	assert.True(t, types["process"])
	assert.Equal(t, "my-service", attributes["service"])
	assert.Equal(t, "my-host", attributes["host"])
	assert.Equal(t, "1234", attributes["process"])
}

func TestEntityMap_All_Empty(t *testing.T) {
	em := NewEntityMap()

	count := 0
	for range em.All() {
		count++
	}

	assert.Equal(t, 0, count)
}

func TestEntityMap_All_EarlyBreak(t *testing.T) {
	em := NewEntityMap()
	em.PutEmpty("service")
	em.PutEmpty("host")
	em.PutEmpty("process")

	count := 0
	for range em.All() {
		count++
		if count == 2 {
			break
		}
	}

	assert.Equal(t, 2, count)
}
