// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctraceerror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTraces(t *testing.T) {
	td := testdata.GenerateTraces(1)
	err := errors.New("some error")
	traceErr := NewTraces(err, td)
	assert.Equal(t, err.Error(), traceErr.Error())
	var target Traces
	assert.False(t, errors.As(nil, &target))
	assert.False(t, errors.As(err, &target))
	assert.True(t, errors.As(traceErr, &target))
	assert.Equal(t, td, target.Data())
}

func TestTraces_Unwrap(t *testing.T) {
	td := testdata.GenerateTraces(1)
	var err error = testErrorType{"some error"}
	// Wrapping err with error Traces.
	traceErr := NewTraces(err, td)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	// Unwrapping traceErr for err and assigning to target.
	require.True(t, errors.As(traceErr, &target))
	require.Equal(t, err, target)
}

type testErrorType struct {
	s string
}

func (t testErrorType) Error() string {
	return ""
}
