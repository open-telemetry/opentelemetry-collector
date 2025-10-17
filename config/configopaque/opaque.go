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

// OpaquePair is a string name/value pair, where the value is opaque.
type OpaquePair = maplist.Pair[String]

// MapList is equivalent to []OpaquePair,
// but can additionally be unmarshalled from a map.
//
// Config validation enforces unicity of keys.
type MapList = maplist.MapList[String]

// MapListFromMap converts a map[string]String to a MapList.
// The output pairs are sorted by name to allow comparisons in tests.
func MapListFromMap(m map[string]String) MapList {
	return maplist.FromMap(m)
}
