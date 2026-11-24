// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filter // import "go.opentelemetry.io/collector/filter"

import "go.opentelemetry.io/collector/strmatch"

// Config configures the matching behavior of a Filter.
//
// Deprecated: [v0.166.0] Use go.opentelemetry.io/collector/strmatch.Config instead.
//
//go:fix inline
type Config = strmatch.Config

// CreateFilter creates a Filter out of a set of Config configuration objects.
//
// Deprecated: [v0.167.0] Use go.opentelemetry.io/collector/strmatch.New instead.
//
//go:fix inline
func CreateFilter(configs []strmatch.Config) strmatch.Matcher {
	return strmatch.New(configs)
}
