// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTraces(t *testing.T) {
	td := testdata.GenerateTraces(1)
	err := errors.New("some error")
	traceErr := NewRetryableTraces(err, td)
	var target RetryableTraces
	assert.False(t, errors.As(nil, &target))
	assert.False(t, errors.As(err, &target))
	assert.True(t, errors.As(traceErr, &target))
	assert.Equal(t, td, target.Data())
}

func TestTraces_Unwrap(t *testing.T) {
	td := testdata.GenerateTraces(1)
	var err error = testErrorType{"some error"}
	// Wrapping err with error Traces.
	traceErr := NewRetryableTraces(err, td)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	// Unwrapping traceErr for err and assigning to target.
	require.True(t, errors.As(traceErr, &target))
	require.Equal(t, err, target)
}

func TestTraces_ShouldIncludeDelay(t *testing.T) {
	td := testdata.GenerateTraces(1)
	var err error = testErrorType{"some error"}

	traceErr, ok := NewRetryableTraces(err, td).(RetryableTraces)
	require.True(t, ok, "Error isn't a retryable error")
	require.ErrorContains(t, traceErr, traceErr.Delay().String())
}

func TestTraces_ShouldIncludeWrappedError(t *testing.T) {
	td := testdata.GenerateTraces(1)
	var err error = testErrorType{"some error"}

	traceErr, ok := NewRetryableTraces(err, td).(RetryableTraces)
	require.True(t, ok, "Error isn't a retryable error")
	require.ErrorContains(t, traceErr, traceErr.Unwrap().Error())
}

func TestLogs(t *testing.T) {
	td := testdata.GenerateLogs(1)
	err := errors.New("some error")
	logsErr := NewLogs(err, td)
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

func TestMetrics(t *testing.T) {
	td := testdata.GenerateMetrics(1)
	err := errors.New("some error")
	metricErr := NewMetrics(err, td)
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
