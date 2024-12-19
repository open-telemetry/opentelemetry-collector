// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentity // import "go.opentelemetry.io/collector/pdata/pentity"

// MarshalSizer is the interface that groups the basic Marshal and Size methods
type MarshalSizer interface {
	Marshaler
	Sizer
}

// Marshaler marshals Entities into bytes.
type Marshaler interface {
	// MarshalEntities the given Entities into bytes.
	// If the error is not nil, the returned bytes slice cannot be used.
	MarshalEntities(ld Entities) ([]byte, error)
}

// Unmarshaler unmarshalls bytes into Entities.
type Unmarshaler interface {
	// UnmarshalEntities the given bytes into Entities.
	// If the error is not nil, the returned Entities cannot be used.
	UnmarshalEntities(buf []byte) (Entities, error)
}

// Sizer is an optional interface implemented by the Marshaler,
// that calculates the size of a marshaled Entities.
type Sizer interface {
	// EntitiesSize returns the size in bytes of a marshaled Entities.
	EntitiesSize(ld Entities) int
}
