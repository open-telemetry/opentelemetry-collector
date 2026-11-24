// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package strmatch // import "go.opentelemetry.io/collector/strmatch"

// Matcher matches values against a set of configured patterns.
type Matcher interface {
	// Matches returns true if the given value matches at least
	// one of the given patterns
	Matches(any) bool
}
