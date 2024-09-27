// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import "go.opentelemetry.io/collector/pdata/pentity"

// NewTextEntitiesMarshaler returns a pentity.Marshaler to encode to OTLP text bytes.
func NewTextEntitiesMarshaler() pentity.Marshaler {
	return textEntitiesMarshaler{}
}

type textEntitiesMarshaler struct{}

// MarshalEntities pentity.Entities to OTLP text.
func (textEntitiesMarshaler) MarshalEntities(ld pentity.Entities) ([]byte, error) {
	buf := dataBuffer{}
	rls := ld.ResourceEntities()
	for i := 0; i < rls.Len(); i++ {
		buf.logEntry("ResourceEntity #%d", i)
		rl := rls.At(i)
		buf.logEntry("Resource SchemaURL: %s", rl.SchemaUrl())
		marshalResource(rl.Resource(), &buf)
		ills := rl.ScopeEntities()
		for j := 0; j < ills.Len(); j++ {
			buf.logEntry("ScopeEntities #%d", j)
			ils := ills.At(j)
			buf.logEntry("ScopeEntities SchemaURL: %s", ils.SchemaUrl())
			buf.logInstrumentationScope(ils.Scope())

			logs := ils.EntityEvents()
			for k := 0; k < logs.Len(); k++ {
				buf.logEntry("EntityEvent #%d", k)
				e := logs.At(k)
				buf.logEntry("EntityType: %s", e.EntityType())
				buf.logEntry("Timestamp: %s", e.Timestamp())
				buf.logAttributes("IDAttributes", e.Id())
				buf.logEntry("Event Type: %s", e.Type())
				if e.Type() == pentity.EventTypeEntityState {
					buf.logAttributes("Attributes", e.EntityState().Attributes())
				}
			}
		}
	}

	return buf.buf.Bytes(), nil
}
