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

type testErrorType struct {
	s string
}

func (t testErrorType) Error() string {
	return ""
}

func TestIsPermanent(t *testing.T) {
	var err error
	assert.False(t, IsPermanent(err))

	err = errors.New("testError")
	assert.False(t, IsPermanent(err))

	err = NewPermanent(err)
	assert.True(t, IsPermanent(err))

	err = fmt.Errorf("%w", err)
	assert.True(t, IsPermanent(err))
}

func TestPermanent_Unwrap(t *testing.T) {
	var err error = testErrorType{"testError"}
	require.False(t, IsPermanent(err))

	// Wrapping testErrorType err with permanent error.
	permanentErr := NewPermanent(err)
	require.True(t, IsPermanent(permanentErr))

	target := testErrorType{}
	require.NotEqual(t, err, target)

	isTestErrorTypeWrapped := errors.As(permanentErr, &target)
	require.True(t, isTestErrorTypeWrapped)

	require.Equal(t, err, target)
}
