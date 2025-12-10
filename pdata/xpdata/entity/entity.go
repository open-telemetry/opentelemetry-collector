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

func (e Entity) Type() string {
	return e.ref.Type()
}

func (e Entity) SchemaURL() string {
	return e.ref.SchemaUrl()
}

func (e Entity) SetSchemaURL(schemaURL string) {
	e.ref.SetSchemaUrl(schemaURL)
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
