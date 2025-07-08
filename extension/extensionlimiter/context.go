// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
	"sync"

	"go.uber.org/multierr"
)

// contextKey is a private type for context keys
type contextKey int

const (
	limiterTrackingKey contextKey = iota
)

// LimiterTrackingState tracks applied limiters by counting requests for each weight key.
// This prevents double-counting across middleware and consumer configurations.
type LimiterTrackingState struct {
	// Counters for each weight key - if non-zero, limiter was already applied
	RequestCount int // WeightKeyRequestCount
	RequestBytes int // WeightKeyRequestBytes
	NetworkBytes int // WeightKeyNetworkBytes
	RequestItems int // WeightKeyRequestItems

	// Compression state for future middleware ordering
	compressionActive bool

	// Rate limiter error state for gRPC stats handlers
	// Allows rate limiters to return errors in the correct context
	rateLimiterMu  sync.Mutex
	rateLimiterErr error
}

// GetLimiterTracking retrieves limiter tracking state from context.
// Returns nil if no tracking state exists.
func GetLimiterTracking(ctx context.Context) *LimiterTrackingState {
	if state, ok := ctx.Value(limiterTrackingKey).(*LimiterTrackingState); ok {
		return state
	}
	return nil
}

// WithLimiterTracking creates a new context with limiter tracking state.
// If tracking state already exists, returns the context unchanged.
func WithLimiterTracking(ctx context.Context) context.Context {
	newCtx, _ := EnsureLimiterTracking(ctx)
	return newCtx
}

// EnsureLimiterTracking ensures limiter tracking state exists in the context
// and returns it. If tracking doesn't exist, it creates new tracking state
// and returns the updated context along with the tracking state.
func EnsureLimiterTracking(ctx context.Context) (context.Context, *LimiterTrackingState) {
	if state := GetLimiterTracking(ctx); state != nil {
		return ctx, state
	}

	state := &LimiterTrackingState{}
	newCtx := context.WithValue(ctx, limiterTrackingKey, state)
	return newCtx, state
}

// AddRequest increments the counter for the given weight key and returns
// true if this is the first request for this weight key (no previous limiter applied).
func (s *LimiterTrackingState) AddRequest(key WeightKey, amount int) bool {
	switch key {
	case WeightKeyRequestCount:
		first := s.RequestCount == 0
		s.RequestCount += amount
		return first
	case WeightKeyRequestBytes:
		first := s.RequestBytes == 0
		s.RequestBytes += amount
		return first
	case WeightKeyNetworkBytes:
		first := s.NetworkBytes == 0
		s.NetworkBytes += amount
		return first
	case WeightKeyRequestItems:
		first := s.RequestItems == 0
		s.RequestItems += amount
		return first
	default:
		return true // Unknown weight key, allow
	}
}

// HasBeenLimited returns true if any limiter has been applied for the given weight key.
func (s *LimiterTrackingState) HasBeenLimited(key WeightKey) bool {
	switch key {
	case WeightKeyRequestCount:
		return s.RequestCount > 0
	case WeightKeyRequestBytes:
		return s.RequestBytes > 0
	case WeightKeyNetworkBytes:
		return s.NetworkBytes > 0
	case WeightKeyRequestItems:
		return s.RequestItems > 0
	default:
		return false
	}
}

// SetCompressionActive marks compression as active or inactive in the context.
func (s *LimiterTrackingState) SetCompressionActive(active bool) {
	s.compressionActive = active
}

// IsCompressionActive returns whether compression is currently active.
func (s *LimiterTrackingState) IsCompressionActive() bool {
	return s.compressionActive
}

// SetRateLimiterError sets a rate limiter error in the tracking state.
func (s *LimiterTrackingState) SetRateLimiterError(err error) {
	s.rateLimiterMu.Lock()
	defer s.rateLimiterMu.Unlock()
	s.rateLimiterErr = multierr.Append(s.rateLimiterErr, err)
}

// GetRateLimiterError retrieves the current rate limiter error without clearing it.
func (s *LimiterTrackingState) GetRateLimiterError() error {
	s.rateLimiterMu.Lock()
	defer s.rateLimiterMu.Unlock()
	return s.rateLimiterErr
}
