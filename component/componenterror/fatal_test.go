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

func TestIsFatal(t *testing.T) {
	var err error
	assert.False(t, IsFatal(err))

	err = errors.New("testError")
	assert.False(t, IsFatal(err))

	err = NewFatal(err)
	assert.True(t, IsFatal(err))

	err = fmt.Errorf("%w", err)
	assert.True(t, IsFatal(err))
}

func TestFatal_Unwrap(t *testing.T) {
	var err error = testErrorType{"testError"}
	require.False(t, IsFatal(err))

	// Wrapping testErrorType err with fatal error.
	fatalErr := NewFatal(err)
	require.True(t, IsFatal(fatalErr))

	target := testErrorType{}
	require.NotEqual(t, err, target)

	isTestErrorTypeWrapped := errors.As(fatalErr, &target)
	require.True(t, isTestErrorTypeWrapped)

	require.Equal(t, err, target)
}
