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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTraces(t *testing.T) {
	td := testdata.GenerateTracesOneSpan()
	err := fmt.Errorf("some error")
	traceErr := NewTraces(err, td)
	assert.Equal(t, err.Error(), traceErr.Error())
	var target Traces
	assert.False(t, AsTraces(nil, &target))
	assert.False(t, AsTraces(err, &target))
	assert.True(t, AsTraces(traceErr, &target))
	assert.Equal(t, td, target.GetTraces())
}

func TestTraces_Unwrap(t *testing.T) {
	td := testdata.GenerateTracesOneSpan()
	var err error = testErrorType{"some error"}
	// Wrapping err with error Traces.
	traceErr := NewTraces(err, td)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	// Unwrapping traceErr for err and assigning to target.
	require.True(t, errors.As(traceErr, &target))
	require.Equal(t, err, target)
}

func TestLogs(t *testing.T) {
	td := testdata.GenerateLogsOneLogRecord()
	err := fmt.Errorf("some error")
	logsErr := NewLogs(err, td)
	assert.Equal(t, err.Error(), logsErr.Error())
	var target Logs
	assert.False(t, AsLogs(nil, &target))
	assert.False(t, AsLogs(err, &target))
	assert.True(t, AsLogs(logsErr, &target))
	assert.Equal(t, td, target.GetLogs())
}

func TestLogs_Unwrap(t *testing.T) {
	td := testdata.GenerateLogsOneLogRecord()
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
	td := testdata.GenerateMetricsOneMetric()
	err := fmt.Errorf("some error")
	metricErr := NewMetrics(err, td)
	assert.Equal(t, err.Error(), metricErr.Error())
	var target Metrics
	assert.False(t, AsMetrics(nil, &target))
	assert.False(t, AsMetrics(err, &target))
	assert.True(t, AsMetrics(metricErr, &target))
	assert.Equal(t, td, target.GetMetrics())
}

func TestMetrics_Unwrap(t *testing.T) {
	td := testdata.GenerateMetricsOneMetric()
	var err error = testErrorType{"some error"}
	// Wrapping err with error Metrics.
	metricErr := NewMetrics(err, td)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	// Unwrapping metricErr for err and assigning to target.
	require.True(t, errors.As(metricErr, &target))
	require.Equal(t, err, target)
}
