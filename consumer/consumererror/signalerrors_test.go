// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

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
	require.EqualError(t, err, traceErr.Error())
	var target Traces
	assert.False(t, errors.As(nil, &target))
	assert.False(t, errors.As(err, &target))
	require.ErrorAs(t, traceErr, &target)
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
	require.ErrorAs(t, traceErr, &target)
	require.Equal(t, err, target)
}

func TestLogs(t *testing.T) {
	td := testdata.GenerateLogs(1)
	err := errors.New("some error")
	logsErr := NewLogs(err, td)
	require.EqualError(t, err, logsErr.Error())
	var target Logs
	assert.False(t, errors.As(nil, &target))
	assert.False(t, errors.As(err, &target))
	require.ErrorAs(t, logsErr, &target)
	assert.Equal(t, td, target.Data())
}

func TestLogs_Unwrap(t *testing.T) {
	td := testdata.GenerateLogs(1)
	var err error = testErrorType{"some error"}
	// Wrapping err with error Logs.
	logsErr := NewLogs(err, td)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	// Unwrapping logsErr for err and assigning to target.
	require.ErrorAs(t, logsErr, &target)
	require.Equal(t, err, target)
}

func TestMetrics(t *testing.T) {
	td := testdata.GenerateMetrics(1)
	err := errors.New("some error")
	metricErr := NewMetrics(err, td)
	require.EqualError(t, err, metricErr.Error())
	var target Metrics
	assert.False(t, errors.As(nil, &target))
	assert.False(t, errors.As(err, &target))
	require.ErrorAs(t, metricErr, &target)
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
	require.ErrorAs(t, metricErr, &target)
	require.Equal(t, err, target)
}
