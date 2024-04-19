// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartial_Error(t *testing.T) {
	e := errors.New("couldn't export")
	pe := NewPartial(e, 3)
	require.Contains(t, pe.Error(), e.Error())
}

func TestPartial_Unwrap(t *testing.T) {
	var err error = testErrorType{"some error"}
	se := NewPartial(err, 4)
	joinedErr := errors.Join(errors.New("other error"), se)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	require.True(t, errors.As(joinedErr, &target))
	require.Equal(t, err, target)
}

func TestPartial_Count(t *testing.T) {
	count := 4
	e := NewPartial(errors.New("couldn't export"), count)
	var pe Partial
	ok := errors.As(e, &pe)
	require.True(t, ok)
	require.Equal(t, count, pe.Count())
}
