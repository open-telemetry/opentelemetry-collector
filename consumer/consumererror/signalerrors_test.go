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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTraceError(t *testing.T) {
	td := testdata.GenerateTraceDataOneSpan()
	err := fmt.Errorf("some error")
	traceErr := Traces(err, td)
	assert.True(t, IsTrace(traceErr))
	assert.Equal(t, err.Error(), traceErr.Error())
	assert.Equal(t, td, GetTraces(traceErr))
}

func TestLogError(t *testing.T) {
	td := testdata.GenerateLogDataOneLog()
	err := fmt.Errorf("some error")
	logsErr := Logs(err, td)
	assert.True(t, IsLogs(logsErr))
	assert.Equal(t, err.Error(), logsErr.Error())
	assert.Equal(t, td, GetLogs(logsErr))
}

func TestMetricError(t *testing.T) {
	td := testdata.GenerateMetricsOneMetric()
	err := fmt.Errorf("some error")
	metricErr := Metrics(err, td)
	assert.True(t, IsMetrics(metricErr))
	assert.Equal(t, err.Error(), metricErr.Error())
	assert.Equal(t, td, GetMetrics(metricErr))
}

func TestPredicates(t *testing.T) {
	for _, tc := range []struct {
		name      string
		expected  bool
		err       error
		predicate func(error) bool
	}{
		{
			name:      "IsNotTraces(nil)",
			expected:  false,
			err:       nil,
			predicate: IsTrace,
		},
		{
			name:      "IsNotTraces",
			expected:  false,
			err:       fmt.Errorf("some error"),
			predicate: IsTrace,
		},
		{
			name:      "IsTraces",
			expected:  true,
			err:       Traces(fmt.Errorf("some error"), testdata.GenerateTraceDataOneSpan()),
			predicate: IsTrace,
		},
		{
			name:      "IsNotMetrics(nil)",
			expected:  false,
			err:       nil,
			predicate: IsMetrics,
		},
		{
			name:      "IsNotMetrics",
			expected:  false,
			err:       fmt.Errorf("some error"),
			predicate: IsMetrics,
		},
		{
			name:      "IsMetrics",
			expected:  true,
			err:       Metrics(fmt.Errorf("some error"), testdata.GenerateMetricsOneMetric()),
			predicate: IsMetrics,
		},
		{
			name:      "IsNotLogs(nil)",
			expected:  false,
			err:       nil,
			predicate: IsLogs,
		},
		{
			name:      "IsNotLogs",
			expected:  false,
			err:       fmt.Errorf("some error"),
			predicate: IsLogs,
		},
		{
			name:      "IsLogs",
			expected:  true,
			err:       Logs(fmt.Errorf("some error"), testdata.GenerateLogDataOneLog()),
			predicate: IsLogs,
		},
		{
			name:      "IsPartial",
			expected:  true,
			err:       Logs(fmt.Errorf("some error"), testdata.GenerateLogDataOneLog()),
			predicate: IsPartial,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.predicate(tc.err))
		})
	}
}
