// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// MultiChecker returns MustDeny when any element returns MustDeny.
type MultiChecker []extensionlimiter.Checker

var _ extensionlimiter.Checker = MultiChecker{}

// MustDeny implements Checker.
func (ls MultiChecker) MustDeny(ctx context.Context) error {
	var err error
	for _, lim := range ls {
		if lim == nil {
			continue
		}
		err = errors.Join(err, lim.MustDeny(ctx))
	}
	return err
}
