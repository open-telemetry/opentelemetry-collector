// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// UnresolvedReason explains why an entity's Global Identity could not be resolved.
type UnresolvedReason int

const (
	// Resolved means the identity was fully resolved.
	Resolved UnresolvedReason = iota
	// UnresolvedScopeUnspecified means an entity in the chain has IdScopeUnspecified.
	UnresolvedScopeUnspecified
	// UnresolvedScopeEntityTypeEmpty means an entity in the chain is IdScopeLocal with an empty scope entity type.
	UnresolvedScopeEntityTypeEmpty
	// UnresolvedScopeEntityMissing means the scoping entity of an entity in the chain is not on the Resource.
	UnresolvedScopeEntityMissing
	// UnresolvedCyclic means the scope chain revisits an entity type.
	UnresolvedCyclic
	// UnresolvedNoSuchEntity means the requested entity type is not on the Resource.
	UnresolvedNoSuchEntity
	// UnresolvedIdentityIncomplete means an entity in the chain references an identifying
	// attribute key that is missing from the Resource attributes, or has no identifying keys.
	UnresolvedIdentityIncomplete
	// UnresolvedScopeEntityTypeUnexpected means an entity in the chain has a scope entity type
	// but its ID scope is not local, which does not conform to the specification.
	UnresolvedScopeEntityTypeUnexpected
	// UnresolvedScopeUnknown means an entity in the chain has an ID scope value this
	// implementation does not know.
	UnresolvedScopeUnknown
)

func (r UnresolvedReason) String() string {
	switch r {
	case Resolved:
		return "resolved"
	case UnresolvedScopeUnspecified:
		return "id scope unspecified"
	case UnresolvedScopeEntityTypeEmpty:
		return "local id scope with empty scope entity type"
	case UnresolvedScopeEntityMissing:
		return "scope entity missing from resource"
	case UnresolvedCyclic:
		return "cyclic scope chain"
	case UnresolvedNoSuchEntity:
		return "entity not on resource"
	case UnresolvedIdentityIncomplete:
		return "identifying attribute missing from resource"
	case UnresolvedScopeEntityTypeUnexpected:
		return "scope entity type set without local id scope"
	case UnresolvedScopeUnknown:
		return "unknown id scope"
	}
	return ""
}

// GlobalIdentity holds the result of composing an entity's Global Identity from a Resource.
type GlobalIdentity struct {
	// Attributes is the union of the Local Identity of the entity and of every entity in its
	// scope chain. When Reason is not Resolved it holds the attributes accumulated before the
	// chain broke, which are usable only within the Resource they came from.
	Attributes pcommon.Map
	// Chain lists the entity types walked, starting with the requested entity.
	Chain []string
	// Reason is Resolved when the identity is a Global Identity, otherwise it says why not.
	Reason UnresolvedReason
}

// Resolved reports whether the identity is a Global Identity.
func (g GlobalIdentity) Resolved() bool {
	return g.Reason == Resolved
}

// ResolveGlobalIdentity composes the Global Identity of the entity of the given type on the
// Resource, following the Identity Scope rules of the entities data model:
//
//  1. Start with the entity's Local Identity.
//  2. If the ID Scope is unspecified, the identity is unresolved.
//  3. If the ID Scope is global, stop.
//  4. If the Scope Entity Type is empty, the identity is unresolved.
//  5. Find the entity on the Resource whose type matches the Scope Entity Type. If there is
//     none, the identity is unresolved.
//  6. Union that entity's Local Identity and repeat from step 2 for it.
//
// A chain that revisits an entity type is cyclic and unresolved.
func ResolveGlobalIdentity(res pcommon.Resource, entityType string) GlobalIdentity {
	entities := ResourceEntities(res)
	out := GlobalIdentity{Attributes: pcommon.NewMap()}
	visited := map[string]bool{}

	current, ok := entities.Get(entityType)
	if !ok {
		out.Reason = UnresolvedNoSuchEntity
		return out
	}

	for {
		t := current.Type()
		if visited[t] {
			out.Reason = UnresolvedCyclic
			return out
		}
		visited[t] = true
		out.Chain = append(out.Chain, t)
		idKeys := current.ref.IdKeys()
		if idKeys.Len() == 0 {
			out.Reason = UnresolvedIdentityIncomplete
			return out
		}
		for _, k := range idKeys.All() {
			v, ok := res.Attributes().Get(k)
			if !ok {
				out.Reason = UnresolvedIdentityIncomplete
				return out
			}
			v.CopyTo(out.Attributes.PutEmpty(k))
		}

		switch current.IDScope() {
		case IdScopeGlobal:
			if current.IDScopeEntityType() != "" {
				out.Reason = UnresolvedScopeEntityTypeUnexpected
				return out
			}
			out.Reason = Resolved
			return out
		case IdScopeLocal:
			scopeType := current.IDScopeEntityType()
			if scopeType == "" {
				out.Reason = UnresolvedScopeEntityTypeEmpty
				return out
			}
			next, found := entities.Get(scopeType)
			if !found {
				out.Reason = UnresolvedScopeEntityMissing
				return out
			}
			current = next
		case IdScopeUnspecified:
			if current.IDScopeEntityType() != "" {
				out.Reason = UnresolvedScopeEntityTypeUnexpected
				return out
			}
			out.Reason = UnresolvedScopeUnspecified
			return out
		default:
			out.Reason = UnresolvedScopeUnknown
			return out
		}
	}
}
