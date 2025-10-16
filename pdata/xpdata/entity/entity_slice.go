// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// EntitySlice logically represents a slice of Entity.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewEntitySlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type EntitySlice struct {
	refs       EntityRefSlice
	attributes pcommon.Map
}

// NewEntitySlice creates a EntitySlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewEntitySlice() EntitySlice {
	return EntitySlice{
		refs:       NewEntityRefSlice(),
		attributes: pcommon.NewMap(),
	}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewEntitySlice()".
func (es EntitySlice) Len() int {
	return es.refs.Len()
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//
//	for i := 0; i < es.Len(); i++ {
//	    e := es.At(i)
//	    ... // Do something with the element
//	}
func (es EntitySlice) At(i int) Entity {
	return Entity{
		ref:        es.refs.At(i),
		attributes: es.attributes,
	}
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new EntitySlice can be initialized:
//
//	es := NewEntitySlice()
//	es.EnsureCapacity(4)
//	for i := 0; i < 4; i++ {
//	    e := es.AppendEmpty()
//	    // Here should set all the values for e.
//	}
func (es EntitySlice) EnsureCapacity(newCap int) {
	es.refs.EnsureCapacity(newCap)
}

// AppendEmpty will append to the end of the slice an empty Entity with specified type.
// It returns the newly added Entity and an error if an entity with the given type already exists.
func (es EntitySlice) AppendEmpty(entityType string) (Entity, error) {
	if entityType == "" {
		return Entity{}, ErrEmptyEntityType
	}
	if es.hasEntityType(entityType) {
		return Entity{}, ErrDuplicateEntityType
	}
	ref := es.refs.AppendEmpty()
	ref.SetType(entityType)
	return Entity{
		ref:        ref,
		attributes: es.attributes,
	}, nil
}

// hasEntityType checks if an entity with the given type already exists in the slice.
func (es EntitySlice) hasEntityType(entityType string) bool {
	if entityType == "" {
		return false
	}
	for i := 0; i < es.Len(); i++ {
		if es.At(i).Type() == entityType {
			return true
		}
	}
	return false
}
