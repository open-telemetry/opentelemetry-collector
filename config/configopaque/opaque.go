// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configopaque // import "go.opentelemetry.io/collector/config/configopaque"

import (
	"fmt"

	"go.opentelemetry.io/collector/internal/maplist"
)

// String alias that is marshaled and printed in an opaque way.
// To recover the original value, cast it to a string.
type String string

const maskedString = "[REDACTED]"

// MarshalText marshals the string as `[REDACTED]`.
func (s String) MarshalText() ([]byte, error) {
	return []byte(maskedString), nil
}

// String formats the string as `[REDACTED]`.
// This is used for the %s and %q verbs.
func (s String) String() string {
	return maskedString
}

// GoString formats the string as `[REDACTED]`.
// This is used for the %#v verb.
func (s String) GoString() string {
	return fmt.Sprintf("%#v", maskedString)
}

// MarshalBinary marshals the string `[REDACTED]` as []byte.
func (s String) MarshalBinary() (text []byte, err error) {
	return []byte(maskedString), nil
}

// *MapList is a replacement for map[string]configopaque.String with a similar API, which can also be unmarshalled from (and is stored as) a list of name/value pairs.
//
// Pairs are assumed to have distinct names. This is checked during config validation.
//
// Similar to native maps, a nil *MapList is treated the same as an empty one for read operations, but write operations will panic.
type MapList = maplist.MapList[String]

// OpaquePair is an element of a MapList.
type OpaquePair = maplist.Pair[String]

// NewMapList is the MapList equivalent of `make(map[string]configopaque.String)`.
func NewMapList() *MapList {
	return maplist.New[String]()
}

// MapListWithCapacity is the MapList equivalent of `make(map[string]configopaque.String, cap)`.
func MapListWithCapacity(capacity int) *MapList {
	return maplist.WithCapacity[String](capacity)
}

// MapListFromMap converts a map[string]configopaque.String to a new *MapList.
// The resulting pairs are stored to facilitate comparisons in tests.
func MapListFromMap(m map[string]String) *MapList {
	return maplist.FromMap(m)
}
