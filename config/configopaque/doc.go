// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configopaque implements String type alias to mask sensitive information.
// Use configopaque.String on the type of sensitive fields, to mask the
// opaque string as `[REDACTED]`.
//
// This ensure that no sensitive information is leaked when printing the
// full Collector configurations.
package configopaque // import "go.opentelemetry.io/collector/config/configopaque"
