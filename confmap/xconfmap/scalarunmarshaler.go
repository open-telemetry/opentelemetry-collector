// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap // import "go.opentelemetry.io/collector/confmap/xconfmap"

import "go.opentelemetry.io/collector/confmap/internal"

// ScalarUnmarshaler is an interface which may be implemented by wrapper types
// to customize their behavior when the type under the wrapper is a scalar
// value.
//
// This should be used for types like `Wrapper[T]` where T is a scalar type, and
// the wrapper type needs to implement custom logic for unmarshaling from a
// scalar value (e.g. `5` for `Wrapper[int]`) into the wrapper type (e.g.
// `Wrapper[int]{inner: 5}`).
type ScalarUnmarshaler = internal.ScalarUnmarshaler

// ScalarValue provides access to an unmarshaled scalar value and allows
// calling back into the confmap decoding/encoding machinery.
type ScalarValue = internal.ScalarValue
