// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

// AnyProvider is an any limiter implementation, possibly one of the
// limiter interfaces.  This serves as a marker for implementations
// which provider rate limiters of any (one)kind.
type AnyProvider interface {
	// unexportedProviderFunc may be embedded using an AnyProviderImpl.
	unexportedProviderFunc()
}

// AnyProviderImpl can be embedded as a marker that a component
// implements one of the rate limiters.
type AnyProviderImpl struct {}

func (AnyProviderImpl) unexportedProviderFunc() {}

