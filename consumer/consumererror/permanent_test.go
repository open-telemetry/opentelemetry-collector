// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestIsPermanent(t *testing.T) {
	tests := []struct {
		err       error
		permanent bool
	}{
		{
			err:       NewPermanent(errors.New("testError")),
			permanent: true,
		},
		{
			err:       NewHTTP(errors.New("testError"), 500),
			permanent: true,
		},
		{
			err:       NewGRPC(errors.New("testError"), nil),
			permanent: true,
		},
		{
			err:       NewPartial(errors.New("testError"), 123),
			permanent: true,
		},
		{
			err:       fmt.Errorf("%w", NewPermanent(errors.New("testError"))),
			permanent: true,
		},
		{
			err:       errors.New("testError"),
			permanent: false,
		},
		{
			err:       NewTraces(errors.New("testError"), ptrace.Traces{}),
			permanent: false,
		},
		{
			err:       NewTraces(NewPermanent(errors.New("testError")), ptrace.Traces{}),
			permanent: true,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.permanent, IsPermanent(tt.err))
	}
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
