// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// SaturationChecker is for checking when a limit is saturated.  This can be
// called prior to the start of work to check for limiter saturation.
type SaturationChecker interface {
	// CheckSaturation is a request to apply a hard limit. If this
	// returns non-nil, the caller must not begin new work in this
	// context.
	CheckSaturation(context.Context) error
}

// CheckSaturationFunc is a functional way to build CheckSaturation functions.
type CheckSaturationFunc func(context.Context) error

// A CheckSaturation function is a complete SaturationChecker.
var _ SaturationChecker = CheckSaturationFunc(nil)

// CheckSaturation implements SaturationChecker.
func (f CheckSaturationFunc) CheckSaturation(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

// SaturationCheckerProvider is an interface to obtain checkers for a group of
// weight keys.
type SaturationCheckerProvider interface {
	// GetSaturationChecker returns a checker for a group of weight keys.
	GetSaturationChecker(...Option) (SaturationChecker, error)
}

// GetSaturationCheckerFunc is a functional way to construct GetSaturationChecker
// functions, used in limiter providers.
type GetSaturationCheckerFunc func(...Option) (SaturationChecker, error)

// SaturationChecker implements SaturationCheckerProvider.
func (f GetSaturationCheckerFunc) GetSaturationChecker(opts ...Option) (SaturationChecker, error) {
	if f == nil {
		return nil, nil
	}
	return f(opts...)
}
