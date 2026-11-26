// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filter // import "go.opentelemetry.io/collector/filter"

import "go.opentelemetry.io/collector/strmatch"

// Filter is an interface for matching values against a set of filters.
//
// Deprecated: [v0.167.0] Use go.opentelemetry.io/collector/strmatch.Matcher instead.
//
//go:fix inline
type Filter = strmatch.Matcher
