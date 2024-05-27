// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cmetricerror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMetrics(t *testing.T) {
	td := testdata.GenerateMetrics(1)
	err := errors.New("some error")
	metricErr := NewMetrics(err, td)
	assert.Equal(t, err.Error(), metricErr.Error())
	var target Metrics
	assert.False(t, errors.As(nil, &target))
	assert.False(t, errors.As(err, &target))
	assert.True(t, errors.As(metricErr, &target))
	assert.Equal(t, td, target.Data())
}

func TestMetrics_Unwrap(t *testing.T) {
	td := testdata.GenerateMetrics(1)
	var err error = testErrorType{"some error"}
	// Wrapping err with error Metrics.
	metricErr := NewMetrics(err, td)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	// Unwrapping metricErr for err and assigning to target.
	require.True(t, errors.As(metricErr, &target))
	require.Equal(t, err, target)
}

type testErrorType struct {
	s string
}

func (t testErrorType) Error() string {
	return ""
}
