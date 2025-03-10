// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiter // import "go.opentelemetry.io/collector/extension/xextension/limiter"

import "context"

type nopLimiter struct{}

var nopLimiterInstance Limiter = &nopLimiter{}

// NewNop returns a limiter that does nothing.
func NewNop() Limiter {
	return nopLimiterInstance
}

// Acquire implements Limiter.
func (nopLimiter) Acquire(_ context.Context, _ uint64) (ReleaseFunc, error) {
	return func() {}, nil
}
