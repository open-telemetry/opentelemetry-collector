// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap // import "go.opentelemetry.io/collector/confmap/xconfmap"

import "go.opentelemetry.io/collector/confmap/internal"

// ScalarMarshaler is an interface which may be implemented by wrapper types
// to customize their behavior when the type under the wrapper is a scalar value.
type ScalarMarshaler = internal.ScalarMarshaler
