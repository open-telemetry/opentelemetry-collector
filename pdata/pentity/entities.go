// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentity // import "go.opentelemetry.io/collector/pdata/pentity"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/entities/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Entities is the top-level struct that is propagated through the entities pipeline.
// Use NewEntities to create new instance, zero-initialized instance is not valid for use.
type Entities internal.Entities

func newEntities(orig *otlpcollectorlog.ExportEntitiesServiceRequest) Entities {
	state := internal.StateMutable
	return Entities(internal.NewEntities(orig, &state))
}

func (ms Entities) getOrig() *otlpcollectorlog.ExportEntitiesServiceRequest {
	return internal.GetOrigEntities(internal.Entities(ms))
}

func (ms Entities) getState() *internal.State {
	return internal.GetEntitiesState(internal.Entities(ms))
}

// NewEntities creates a new Entities struct.
func NewEntities() Entities {
	return newEntities(&otlpcollectorlog.ExportEntitiesServiceRequest{})
}

// IsReadOnly returns true if this Entities instance is read-only.
func (ms Entities) IsReadOnly() bool {
	return *ms.getState() == internal.StateReadOnly
}

// CopyTo copies the Entities instance overriding the destination.
func (ms Entities) CopyTo(dest Entities) {
	ms.ResourceEntities().CopyTo(dest.ResourceEntities())
}

// EntityCount calculates the total number of entities.
func (ms Entities) EntityCount() int {
	entitiesCount := 0
	for i := 0; i < ms.ResourceEntities().Len(); i++ {
		ses := ms.ResourceEntities().At(i).ScopeEntities()
		for j := 0; j < ses.Len(); j++ {
			entitiesCount += ses.At(i).EntityEvents().Len()
		}
	}
	return entitiesCount
}

// ResourceEntities returns the ResourceEntitiesSlice associated with this Entities.
func (ms Entities) ResourceEntities() ResourceEntitiesSlice {
	return newResourceEntitiesSlice(&ms.getOrig().ResourceEntities, internal.GetEntitiesState(internal.Entities(ms)))
}

// MarkReadOnly marks the Entities as shared so that no further modifications can be done on it.
func (ms Entities) MarkReadOnly() {
	internal.SetEntitiesState(internal.Entities(ms), internal.StateReadOnly)
}

// Entity is an abstraction of an entity that can be converted to/from EntityEvent and ResourceEntityRef.
// Use NewEntity to create new instance, zero-initialized instance is not valid for use.
type Entity struct {
	typ   string
	id    pcommon.Map
	attrs pcommon.Map
}

// NewEntity creates a new Entity.
// id must not be mutated after it is passed to this function.
func NewEntity(typ string, id pcommon.Map, attrs pcommon.Map) Entity {
	return Entity{typ: typ, id: id, attrs: attrs}
}

func (ms Entity) Type() string {
	return ms.typ
}

// ID returns the ID associated with this Entity. The returned map must not be mutated.
func (ms Entity) ID() pcommon.Map {
	return ms.id
}

// Attributes returns the attributes associated with this Entity. The returned map must not be mutated.
func (ms Entity) Attributes() pcommon.Map {
	return ms.attrs
}

// CopyTo copies the Entity instance overriding the destination.
func (ms Entity) CopyTo(dest Entity) {
	dest.typ = ms.typ
	ms.id.CopyTo(dest.id)
	ms.attrs.CopyTo(dest.attrs)
}

// EntityEvent returns the "state" EntityEvent representation of this Entity.
func (ms Entity) EntityEvent(ts pcommon.Timestamp) EntityEvent {
	ee := NewEntityEvent()
	ee.SetTimestamp(ts)
	ee.SetEntityType(ms.typ)
	ms.id.CopyTo(ee.Id())
	ms.attrs.CopyTo(ee.EntityState().Attributes())
	return ee
}

// AddToResource appends this Entity to the Resource as a ResourceEntityRef overriding existing one if any with
// the same type and any conflicting resource attributes.
func (ms Entity) AddToResource(res pcommon.Resource) {
	var er pcommon.ResourceEntityRef
	erFound := false
	for i := 0; i < res.Entities().Len(); i++ {
		if res.Entities().At(i).Type() == ms.typ {
			er = res.Entities().At(i)
			er.IdAttrKeys().FromRaw([]string{})
			er.DescrAttrKeys().FromRaw([]string{})
			erFound = true
			break
		}
	}
	if !erFound {
		er = res.Entities().AppendEmpty()
		er.SetType(ms.typ)
	}
	ms.ID().Range(func(k string, v pcommon.Value) bool {
		er.IdAttrKeys().Append(k)
		v.CopyTo(res.Attributes().PutEmpty(k))
		return true
	})
	ms.Attributes().Range(func(k string, v pcommon.Value) bool {
		er.DescrAttrKeys().Append(k)
		v.CopyTo(res.Attributes().PutEmpty(k))
		return true
	})
}

// GetEntityFromResource returns the Entity from the Resource that matches the type of this Entity.
func GetEntityFromResource(res pcommon.Resource, typ string) (Entity, bool) {
	for i := 0; i < res.Entities().Len(); i++ {
		er := res.Entities().At(i)
		if er.Type() == typ {
			id := pcommon.NewMap()
			for j := 0; j < er.IdAttrKeys().Len(); j++ {
				k := er.IdAttrKeys().At(j)
				if v, ok := res.Attributes().Get(k); ok {
					v.CopyTo(id.PutEmpty(k))
				}
			}
			attrs := pcommon.NewMap()
			for j := 0; j < er.DescrAttrKeys().Len(); j++ {
				k := er.DescrAttrKeys().At(j)
				if v, ok := res.Attributes().Get(k); ok {
					v.CopyTo(attrs.PutEmpty(k))
				}
			}
			return NewEntity(typ, id, attrs), true
		}
	}
	return Entity{}, false
}
