// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"iter"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// EntityMap logically represents a map of Entity keyed by entity type.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewEntityMap function to create new instances.
// Important: zero-initialized instance is not valid for use.
type EntityMap struct {
	refs       EntityRefSlice
	attributes pcommon.Map
}

// NewEntityMap creates a EntityMap with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewEntityMap() EntityMap {
	return EntityMap{
		refs:       NewEntityRefSlice(),
		attributes: pcommon.NewMap(),
	}
}

// Len returns the number of elements in the map.
//
// Returns "0" for a newly instance created with "NewEntityMap()".
func (em EntityMap) Len() int {
	return em.refs.Len()
}

// EnsureCapacity is an operation that ensures the map has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the map capacity will be expanded to equal newCap.
func (em EntityMap) EnsureCapacity(newCap int) {
	em.refs.EnsureCapacity(newCap)
}

// Get returns the Entity associated with the entityType and true. The returned
// Entity is not a copy, it is a reference to the entity stored in this map.
// It is allowed to modify the returned entity.
// Such modification will be applied to the entity stored in this map.
//
// If the entityType does not exist, returns a zero-initialized Entity and false.
// Calling any functions on the returned invalid instance may cause a panic.
func (em EntityMap) Get(entityType string) (Entity, bool) {
	if entityType == "" {
		return Entity{}, false
	}
	for i := 0; i < em.Len(); i++ {
		if em.refs.At(i).Type() == entityType {
			return Entity{
				ref:        em.refs.At(i),
				attributes: em.attributes,
			}, true
		}
	}
	return Entity{}, false
}

// All returns an iterator over entity type-Entity pairs in the EntityMap.
//
//	for entityType, entity := range em.All() {
//	    ... // Do something with entity type and entity
//	}
func (em EntityMap) All() iter.Seq2[string, Entity] {
	return func(yield func(string, Entity) bool) {
		for i := 0; i < em.Len(); i++ {
			ref := em.refs.At(i)
			entity := Entity{
				ref:        ref,
				attributes: em.attributes,
			}
			if !yield(ref.Type(), entity) {
				return
			}
		}
	}
}

// Remove removes the entity associated with the entityType and returns true if the entity
// was present in the map, otherwise returns false. All attributes associated with the entity
// are also removed.
func (em EntityMap) Remove(entityType string) bool {
	for i := 0; i < em.refs.Len(); i++ {
		ref := em.refs.At(i)
		if ref.Type() == entityType {
			for _, k := range ref.IdKeys().All() {
				em.attributes.Remove(k)
			}
			for _, k := range ref.DescriptionKeys().All() {
				em.attributes.Remove(k)
			}
			em.refs.RemoveIf(func(er EntityRef) bool {
				return er.Type() == entityType
			})
			return true
		}
	}
	return false
}

// PutEmpty inserts or replaces an empty Entity with the specified type and returns it.
// If an entity with the given type already exists, it replaces it and removes all attributes associated with it.
func (em EntityMap) PutEmpty(entityType string) Entity {
	em.Remove(entityType)
	ref := em.refs.AppendEmpty()
	ref.SetType(entityType)
	return Entity{
		ref:        ref,
		attributes: em.attributes,
	}
}
