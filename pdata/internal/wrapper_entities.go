// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/entities/v1"
	otlpentities "go.opentelemetry.io/collector/pdata/internal/data/protogen/entities/v1"
)

type Entities struct {
	orig  *otlpcollectorlog.ExportEntitiesServiceRequest
	state *State
}

func GetOrigEntities(ms Entities) *otlpcollectorlog.ExportEntitiesServiceRequest {
	return ms.orig
}

func GetEntitiesState(ms Entities) *State {
	return ms.state
}

func SetEntitiesState(ms Entities, state State) {
	*ms.state = state
}

func NewEntities(orig *otlpcollectorlog.ExportEntitiesServiceRequest, state *State) Entities {
	return Entities{orig: orig, state: state}
}

// EntitiesToProto internal helper to convert Entities to protobuf representation.
func EntitiesToProto(l Entities) otlpentities.EntitiesData {
	return otlpentities.EntitiesData{
		ResourceEntities: l.orig.ResourceEntities,
	}
}

// EntitiesFromProto internal helper to convert protobuf representation to Entities.
// This function set exclusive state assuming that it's called only once per Entities.
func EntitiesFromProto(orig otlpentities.EntitiesData) Entities {
	state := StateMutable
	return NewEntities(&otlpcollectorlog.ExportEntitiesServiceRequest{
		ResourceEntities: orig.ResourceEntities,
	}, &state)
}
