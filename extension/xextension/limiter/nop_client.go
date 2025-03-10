// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiter // import "go.opentelemetry.io/collector/extension/xextension/limiter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

type nopExtension struct {
	extension.Extension
}

var nopExtensionInstance Extension = &nopExtension{}

type nopLimiter struct{}

var nopLimiterInstance Limiter = &nopLimiter{}

// NewNop returns a limiter extension that does nothing.
func NewNop() Extension {
	return nopExtensionInstance
}

// Acquire implements Limiter.
func (nopExtension) GetLimiter(_ context.Context, _ component.Kind, _ component.ID) (Limiter, error) {
	return nopLimiterInstance, nil
}

func (nopLimiter) Acquire(_ context.Context, _ uint64) (ReleaseFunc, error) {
	return func() {}, nil
}
