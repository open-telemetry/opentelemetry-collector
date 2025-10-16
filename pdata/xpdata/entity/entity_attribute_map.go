// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var (
	ErrConflictingAttribute = errors.New("attribute key already exists with a different value")
	ErrEmptyEntityType      = errors.New("entity type cannot be empty")
	ErrDuplicateEntityType  = errors.New("entity with this type already exists")
)

// EntityAttributeMap is a wrapper around pcommon.Map that restricts operations to only the keys
// that belong to a specific entity (either id or description attributes).
type EntityAttributeMap struct {
	keys       pcommon.StringSlice
	attributes pcommon.Map
}

// Get returns the Value associated with the key and true. Returned
// Value is not a copy, it is a reference to the value stored in this map. It is
// allowed to modify the returned value using Value.Set* functions.
//
// If the key does not exist in the entity's key list or in the underlying map,
// returns an invalid instance and false. Calling any functions on the returned
// invalid instance will cause a panic.
func (m EntityAttributeMap) Get(key string) (pcommon.Value, bool) {
	if !m.containsKey(key) {
		return pcommon.Value{}, false
	}
	return m.attributes.Get(key)
}

// PutEmpty inserts or updates an empty value to the map under given key
// and return the updated/inserted value.
// The key is also added to the entity's key list if not already present.
// Returns ErrConflictingAttribute if the key already exists in the shared attributes
// but is not associated with this entity.
func (m EntityAttributeMap) PutEmpty(k string) (pcommon.Value, error) {
	if !m.containsKey(k) {
		if _, ok := m.attributes.Get(k); ok {
			return pcommon.Value{}, fmt.Errorf("%w: %q", ErrConflictingAttribute, k)
		}
		m.keys.Append(k)
	}
	return m.attributes.PutEmpty(k), nil
}

// PutStr performs the Insert or Update action. The Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
// The key is also added to the entity's key list if not already present.
// Returns ErrConflictingAttribute if the key already exists in the shared attributes
// but is not associated with this entity.
func (m EntityAttributeMap) PutStr(k, v string) error {
	if !m.containsKey(k) {
		if _, ok := m.attributes.Get(k); ok {
			return fmt.Errorf("%w: %q", ErrConflictingAttribute, k)
		}
		m.keys.Append(k)
	}
	m.attributes.PutStr(k, v)
	return nil
}

// Remove removes the entry associated with the key and returns true if the key existed.
// The key is also removed from the entity's key list.
func (m EntityAttributeMap) Remove(key string) bool {
	var keyFound bool
	m.keys.RemoveIf(func(k string) bool {
		if k == key {
			keyFound = true
			return true
		}
		return false
	})
	if !keyFound {
		return false
	}
	m.attributes.Remove(key)
	return true
}

func (m EntityAttributeMap) containsKey(key string) bool {
	for _, k := range m.keys.All() {
		if k == key {
			return true
		}
	}
	return false
}
