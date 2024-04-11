// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

type testErrorType struct {
	s string
}

func (t testErrorType) Error() string {
	return ""
}

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

func TestTraces_Delay(t *testing.T) {
	delay := 4 * time.Second
	td := testdata.GenerateTraces(1)
	// Wrapping err with error Traces.
	err := NewTraces(errors.New("some error"), td, WithRetryDelay(delay))
	var traceErr Traces
	// Unwrapping traceErr for err and assigning to target.
	require.True(t, errors.As(err, &traceErr))
	require.Equal(t, traceErr.Delay(), delay)
}

func TestLogs(t *testing.T) {
	td := testdata.GenerateLogs(1)
	err := errors.New("some error")
	logsErr := NewLogs(err, td)
	assert.Equal(t, err.Error(), logsErr.Error())
	var target Logs
	assert.False(t, errors.As(nil, &target))
	assert.False(t, errors.As(err, &target))
	assert.True(t, errors.As(logsErr, &target))
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
	require.True(t, errors.As(logsErr, &target))
	require.Equal(t, err, target)
}

func TestLogs_Delay(t *testing.T) {
	delay := 4 * time.Second
	td := testdata.GenerateLogs(1)
	// Wrapping err with error Logs.
	err := NewLogs(errors.New("some error"), td, WithRetryDelay(delay))
	var logErr Logs
	// Unwrapping LogErr for err and assigning to target.
	require.True(t, errors.As(err, &logErr))
	require.Equal(t, logErr.Delay(), delay)
}

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

func TestMetrics_Delay(t *testing.T) {
	delay := 4 * time.Second
	td := testdata.GenerateMetrics(1)
	// Wrapping err with error Metrics.
	err := NewMetrics(errors.New("some error"), td, WithRetryDelay(delay))
	var metricErr Metrics
	// Unwrapping MetricErr for err and assigning to target.
	require.True(t, errors.As(err, &metricErr))
	require.Equal(t, metricErr.Delay(), delay)
}
