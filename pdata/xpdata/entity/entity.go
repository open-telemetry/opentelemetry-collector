// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Entity is a helper struct that represents an entity in a more user-friendly way than the underlying
// EntityRef protobuf message. After adding an entity to a resource, the entity shares the resource's
// attributes map, so modifications to the entity's attributes are immediately reflected in the resource.
// To create an Entity, use the EntityMap's PutEmpty method.
type Entity struct {
	ref        EntityRef
	attributes pcommon.Map
}

func NewEntity(t string) Entity {
	ref := NewEntityRef()
	ref.SetType(t)
	return Entity{
		ref:        ref,
		attributes: pcommon.NewMap(),
	}
}

func (e Entity) Type() string {
	return e.ref.Type()
}

func (e Entity) SchemaURL() string {
	return e.ref.SchemaUrl()
}

func (e Entity) SetSchemaURL(schemaURL string) {
	e.ref.SetSchemaUrl(schemaURL)
}

// IDScope returns whether the entity's ID is unique on its own or only within another entity.
func (e Entity) IDScope() IdScope {
	return e.ref.IdScope()
}

// SetIDScope sets whether the entity's ID is unique on its own or only within another entity.
// Only the producer of the entity may set this.
func (e Entity) SetIDScope(scope IdScope) {
	e.ref.SetIdScope(scope)
}

// IDScopeEntityType returns the type of the entity within which this entity's ID is unique.
// It is set only when IDScope is IdScopeLocal.
func (e Entity) IDScopeEntityType() string {
	return e.ref.IdScopeEntityType()
}

// SetIDScopeEntityType sets the type of the entity within which this entity's ID is unique.
// Only the producer of the entity may set this.
func (e Entity) SetIDScopeEntityType(entityType string) {
	e.ref.SetIdScopeEntityType(entityType)
}

// IdentifyingAttributes returns an EntityAttributeMap for managing the entity's identifying attributes.
func (e Entity) IdentifyingAttributes() EntityAttributeMap {
	return EntityAttributeMap{
		keys:       e.ref.IdKeys(),
		attributes: e.attributes,
	}
}

// DescriptiveAttributes returns an EntityAttributeMap for managing the entity's descriptive attributes.
func (e Entity) DescriptiveAttributes() EntityAttributeMap {
	return EntityAttributeMap{
		keys:       e.ref.DescriptionKeys(),
		attributes: e.attributes,
	}
}

// CopyToResource moves the entity to the provided resource by overriding existing entities and attributes.
func (e Entity) CopyToResource(res pcommon.Resource) {
	ent := ResourceEntities(res).PutEmpty(e.Type())
	ent.SetSchemaURL(e.SchemaURL())
	ent.SetIDScope(e.IDScope())
	ent.SetIDScopeEntityType(e.IDScopeEntityType())
	for k, v := range e.IdentifyingAttributes().All() {
		v.CopyTo(ent.IdentifyingAttributes().PutEmpty(k))
	}
	for k, v := range e.DescriptiveAttributes().All() {
		v.CopyTo(ent.DescriptiveAttributes().PutEmpty(k))
	}
}
