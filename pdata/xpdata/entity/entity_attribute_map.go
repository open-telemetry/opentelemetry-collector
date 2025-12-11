// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import "go.opentelemetry.io/collector/pdata/pcommon"

// EntityAttributeMap is a wrapper around pcommon.Map that restricts operations to only the keys
// that belong to a specific set of entity attributes (either ID or Description attributes).
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

// CanPut returns true if it's safe to call Put methods on the given key.
// Returns true if:
//   - The key is already owned by this entity (in the entity's key list), OR
//   - The key doesn't exist in the shared attributes map (available to claim)
//
// Returns false if the key exists in the shared map but belongs to another entity.
//
// Use this method before calling Put* methods to avoid conflicts:
//
//	if entity.IdentifyingAttributes().CanPut("service.name") {
//	    entity.IdentifyingAttributes().PutStr("service.name", "my-service")
//	}
func (m EntityAttributeMap) CanPut(key string) bool {
	if m.containsKey(key) {
		return true
	}
	_, exists := m.attributes.Get(key)
	return !exists
}

// PutEmpty inserts or updates an empty value to the map under given key
// and returns the updated/inserted value.
// The key is also added to the entity's key list if not already present.
//
// WARNING: This method is destructive and will overwrite any existing value in the shared
// attributes map, even if it belongs to another entity. Use CanPut() to check safety first
// if you need to avoid conflicts with other entities.
func (m EntityAttributeMap) PutEmpty(k string) pcommon.Value {
	if !m.containsKey(k) {
		m.keys.Append(k)
	}
	return m.attributes.PutEmpty(k)
}

// PutStr performs the Insert or Update action. The Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
// The key is also added to the entity's key list if not already present.
//
// WARNING: This method is destructive and will overwrite any existing value in the shared
// attributes map, even if it belongs to another entity. Use CanPut() to check safety first
// if you need to avoid conflicts with other entities.
func (m EntityAttributeMap) PutStr(k, v string) {
	if !m.containsKey(k) {
		m.keys.Append(k)
	}
	m.attributes.PutStr(k, v)
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
