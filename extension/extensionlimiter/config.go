// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

// Option is passed to limiter providers.
//
// NOTE: For data-specific or tenant-specific limits we will extend
// providers with Options and add a Config type, but none are
// supported yet and this PR contains only interfaces, not need for
// options in core repository components.
type Option interface {
	apply()
}
