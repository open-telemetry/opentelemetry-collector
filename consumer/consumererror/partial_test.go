// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPartialError(t *testing.T) {
	origErr := errors.New("some items dropped")
	dropped := 5
	err := NewPartialError(origErr, dropped)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some items dropped")
}

func TestPartialError_Unwrap(t *testing.T) {
	origErr := errors.New("original error")
	err := NewPartialError(origErr, 3)

	assert.True(t, errors.Is(err, origErr))
}

func TestAsPartialError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectDropped   int
		expectIsPartial bool
	}{
		{
			name:            "partial error",
			err:             NewPartialError(errors.New("dropped"), 5),
			expectDropped:   5,
			expectIsPartial: true,
		},
		{
			name:            "wrapped partial error",
			err:             fmt.Errorf("wrapped: %w", NewPartialError(errors.New("dropped"), 10)),
			expectDropped:   10,
			expectIsPartial: true,
		},
		{
			name:            "not a partial error",
			err:             errors.New("regular error"),
			expectDropped:   0,
			expectIsPartial: false,
		},
		{
			name:            "nil error",
			err:             nil,
			expectDropped:   0,
			expectIsPartial: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dropped, ok := AsPartialError(tt.err)
			require.Equal(t, tt.expectIsPartial, ok)
			assert.Equal(t, tt.expectDropped, dropped)
		})
	}
}

func TestPartialError_Dropped(t *testing.T) {
	err := NewPartialError(errors.New("test"), 42)
	var p partialError
	require.True(t, errors.As(err, &p))
	assert.Equal(t, 42, p.Dropped())
}
