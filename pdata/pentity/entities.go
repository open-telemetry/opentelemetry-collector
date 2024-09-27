// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentity // import "go.opentelemetry.io/collector/pdata/pentity"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/entities/v1"
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
	ms.ScopeEntities().CopyTo(dest.ScopeEntities())
}

// LogRecordCount calculates the total number of log records.
func (ms Entities) LogRecordCount() int {
	logCount := 0
	ill := ms.ScopeEntities()
	for i := 0; i < ill.Len(); i++ {
		entities := ill.At(i)
		logCount += entities.EntityEvents().Len()
	}
	return logCount
}

// ScopeEntities returns the ScopeEntitiesSlice associated with this Entities.
func (ms Entities) ScopeEntities() ScopeEntitiesSlice {
	return newScopeEntitiesSlice(&ms.getOrig().ScopeEntities, internal.GetEntitiesState(internal.Entities(ms)))
}

// MarkReadOnly marks the Entities as shared so that no further modifications can be done on it.
func (ms Entities) MarkReadOnly() {
	internal.SetEntitiesState(internal.Entities(ms), internal.StateReadOnly)
}
