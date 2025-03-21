// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// Predefined weight keys for common rate limits
const (
	WeightKeyNetworkBytes   = "network_bytes"
	WeightKeyRequestItems   = "request_items"
	WeightKeyRequestCount   = "request_count"
	WeightKeyResidentMemory = "resident_memory"
)

// Weight represents a specific dimension to be limited with its associated value.
type Weight struct {
	// Key identifies the type of weight being limited
	Key string
	// Value is the numeric value to be considered for limiting
	Value uint64
}

// ReleaseFunc is called when resources should be released after limiting.
type ReleaseFunc func()

// Limiter is an interface that components can use to apply rate limiting.
// Extensions implementing this interface can be referenced by their
// names from component rate limiting configurations.
type Limiter interface {
	// Acquire attempts to acquire resources based on the provided weights.
	// It may block until resources are available or return an error if the limit
	// cannot be satisfied. On success, it returns a ReleaseFunc that should be called
	// when the resources are no longer needed.
	Acquire(ctx context.Context, weights []Weight) (ReleaseFunc, error)
}

var _ Limiter = LimiterFunc(nil)

type LimiterFunc func(ctx context.Context, weights []Weight) (ReleaseFunc, error)

func (f LimiterFunc) Acquire(ctx context.Context, weights []Weight) (ReleaseFunc, error) {
	if f == nil {
		return func() {}, nil
	}
	return f(ctx, weights)
}
