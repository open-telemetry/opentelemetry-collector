// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartialReceiveError(t *testing.T) {
	innerErr := errors.New("translation failed for 3 metrics")
	partialErr := NewPartialReceiveError(innerErr, 3)

	assert.Equal(t, "translation failed for 3 metrics", partialErr.Error())
	assert.Equal(t, innerErr, errors.Unwrap(partialErr))
}

func TestIsPartialReceiveError(t *testing.T) {
	innerErr := errors.New("validation error")

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "partial receive error",
			err:      NewPartialReceiveError(innerErr, 5),
			expected: true,
		},
		{
			name:     "wrapped partial receive error",
			err:      fmt.Errorf("wrapped: %w", NewPartialReceiveError(innerErr, 5)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPartialReceiveError(tt.err))
		})
	}
}

func TestFailedItemsFromError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedCount int
		expectedOk    bool
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedCount: 0,
			expectedOk:    false,
		},
		{
			name:          "regular error",
			err:           errors.New("regular error"),
			expectedCount: 0,
			expectedOk:    false,
		},
		{
			name:          "partial receive error with 5 failed",
			err:           NewPartialReceiveError(errors.New("test"), 5),
			expectedCount: 5,
			expectedOk:    true,
		},
		{
			name:          "wrapped partial receive error",
			err:           fmt.Errorf("wrapped: %w", NewPartialReceiveError(errors.New("test"), 10)),
			expectedCount: 10,
			expectedOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, ok := FailedItemsFromError(tt.err)
			assert.Equal(t, tt.expectedCount, count)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

func TestPartialReceiveErrorUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	partialErr := NewPartialReceiveError(innerErr, 7)

	// Test errors.Is
	require.True(t, errors.Is(partialErr, innerErr))

	// Test errors.As
	var target PartialReceiveError
	require.True(t, errors.As(partialErr, &target))
	assert.Equal(t, 7, target.Failed)
}
