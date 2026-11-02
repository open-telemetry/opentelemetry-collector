// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity // import "go.opentelemetry.io/collector/pdata/xpdata/entity"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

// IdScope states whether an entity's ID is unique on its own or only within another entity.
type IdScope int32

const (
	// IdScopeUnspecified means the producer did not state the scope. Consumers must treat
	// the entity's identity as unresolved.
	IdScopeUnspecified = IdScope(internal.IdScope_ID_SCOPE_UNSPECIFIED)
	// IdScopeGlobal means the ID is unique on its own. IdScopeEntityType must be empty.
	IdScopeGlobal = IdScope(internal.IdScope_ID_SCOPE_GLOBAL)
	// IdScopeLocal means the ID is unique only within one other entity, named by
	// IdScopeEntityType, which must be set.
	IdScopeLocal = IdScope(internal.IdScope_ID_SCOPE_LOCAL)
)

// String returns the string representation of the IdScope.
func (s IdScope) String() string {
	switch s {
	case IdScopeUnspecified:
		return "Unspecified"
	case IdScopeGlobal:
		return "Global"
	case IdScopeLocal:
		return "Local"
	}
	return ""
}
