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
	origErr := errors.New("translation failed")

	t.Run("basic", func(t *testing.T) {
		err := NewPartialError(origErr, 5)
		assert.ErrorIs(t, err, origErr)
		assert.Equal(t, "translation failed", err.Error())

		pe := GetPartialError(err)
		require.NotNil(t, pe)
		assert.Equal(t, 5, pe.FailedCount())
	})

	t.Run("zero_count", func(t *testing.T) {
		err := NewPartialError(origErr, 0)
		pe := GetPartialError(err)
		require.NotNil(t, pe)
		assert.Equal(t, 0, pe.FailedCount())
	})

	t.Run("negative_count_becomes_zero", func(t *testing.T) {
		err := NewPartialError(origErr, -1)
		pe := GetPartialError(err)
		require.NotNil(t, pe)
		assert.Equal(t, 0, pe.FailedCount())
	})
}

func TestGetPartialError(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		pe := GetPartialError(nil)
		assert.Nil(t, pe)
	})

	t.Run("non_partial_error", func(t *testing.T) {
		err := errors.New("regular error")
		pe := GetPartialError(err)
		assert.Nil(t, pe)
	})

	t.Run("wrapped_partial_error", func(t *testing.T) {
		origErr := errors.New("inner error")
		partialErr := NewPartialError(origErr, 10)
		wrappedErr := fmt.Errorf("outer: %w", partialErr)

		pe := GetPartialError(wrappedErr)
		require.NotNil(t, pe)
		assert.Equal(t, 10, pe.FailedCount())
	})
}

func TestPartialError_Unwrap(t *testing.T) {
	var innerErr error = testErrorType{"testError"}

	partialErr := NewPartialError(innerErr, 7)
	require.NotNil(t, GetPartialError(partialErr))

	// Verify the inner error can be extracted with errors.As
	target := testErrorType{}
	require.NotEqual(t, innerErr, target)

	isWrapped := errors.As(partialErr, &target)
	require.True(t, isWrapped)
	require.Equal(t, innerErr, target)
}

func TestPartialError_CombinedWithDownstream(t *testing.T) {
	origErr := errors.New("queue full")
	partialErr := NewPartialError(origErr, 30)
	downstreamErr := NewDownstream(partialErr)

	// Should be recognized as downstream error
	assert.True(t, IsDownstream(downstreamErr))

	// Should still be able to extract the partial error
	pe := GetPartialError(downstreamErr)
	require.NotNil(t, pe)
	assert.Equal(t, 30, pe.FailedCount())

	// Original error should still be accessible
	assert.ErrorIs(t, downstreamErr, origErr)
}

func TestPartialError_CombinedWithPermanent(t *testing.T) {
	origErr := errors.New("invalid format")
	partialErr := NewPartialError(origErr, 15)
	permanentErr := NewPermanent(partialErr)

	// Should be recognized as permanent error
	assert.True(t, IsPermanent(permanentErr))

	// Should still be able to extract the partial error
	pe := GetPartialError(permanentErr)
	require.NotNil(t, pe)
	assert.Equal(t, 15, pe.FailedCount())

	// Original error should still be accessible
	assert.ErrorIs(t, permanentErr, origErr)
}

func TestPartialError_DownstreamWrappingPartial(t *testing.T) {
	// Test the recommended pattern: downstream wrapping partial
	origErr := errors.New("pipeline refused")
	partialErr := NewPartialError(origErr, 20)
	combinedErr := NewDownstream(partialErr)

	// All checks should work
	assert.True(t, IsDownstream(combinedErr))
	pe := GetPartialError(combinedErr)
	require.NotNil(t, pe)
	assert.Equal(t, 20, pe.FailedCount())
	assert.ErrorIs(t, combinedErr, origErr)
}
