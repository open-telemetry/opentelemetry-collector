// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenterror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRecoverable(t *testing.T) {
	var err error
	assert.False(t, IsRecoverable(err))

	err = errors.New("testError")
	assert.False(t, IsRecoverable(err))

	err = NewRecoverable(err)
	assert.True(t, IsRecoverable(err))

	err = fmt.Errorf("%w", err)
	assert.True(t, IsRecoverable(err))
}

func TestRecoverable_Unwrap(t *testing.T) {
	var err error = testErrorType{"testError"}
	require.False(t, IsRecoverable(err))

	// Wrapping testErrorType err with recoverable error.
	recoverableErr := NewRecoverable(err)
	require.True(t, IsRecoverable(recoverableErr))

	target := testErrorType{}
	require.NotEqual(t, err, target)

	isTestErrorTypeWrapped := errors.As(recoverableErr, &target)
	require.True(t, isTestErrorTypeWrapped)

	require.Equal(t, err, target)
}
