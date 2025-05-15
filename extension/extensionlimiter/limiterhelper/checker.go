// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// MultiBaseLimiter returns MustDeny when any element returns MustDeny.
type MultiBaseLimiter []extensionlimiter.BaseLimiter

var _ extensionlimiter.BaseLimiter = MultiBaseLimiter{}

// MustDeny implements BaseLimiter.
func (ls MultiBaseLimiter) MustDeny(ctx context.Context) error {
	var err error
	for _, lim := range ls {
		if lim == nil {
			continue
		}
		err = errors.Join(err, lim.MustDeny(ctx))
	}
	return err
}

// NeverDeny returns a BaseLimiter that never denies.
func NeverDeny() extensionlimiter.BaseLimiter {
	return extensionlimiter.MustDenyFunc(nil)
}
