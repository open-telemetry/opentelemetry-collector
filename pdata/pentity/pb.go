// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentity // import "go.opentelemetry.io/collector/pdata/pentity"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpentities "go.opentelemetry.io/collector/pdata/internal/data/protogen/entities/v1"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalEntities(ed Entities) ([]byte, error) {
	pb := internal.EntitiesToProto(internal.Entities(ed))
	return pb.Marshal()
}

func (e *ProtoMarshaler) EntitiesSize(ed Entities) int {
	pb := internal.EntitiesToProto(internal.Entities(ed))
	return pb.Size()
}

var _ Unmarshaler = (*ProtoUnmarshaler)(nil)

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalEntities(buf []byte) (Entities, error) {
	pb := otlpentities.EntitiesData{}
	err := pb.Unmarshal(buf)
	return Entities(internal.EntitiesFromProto(pb)), err
}
