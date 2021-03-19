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

	"go.opentelemetry.io/collector/consumer/pdata"
)

// PartialError can be used to signal that a subset of received data failed to be processed or sent.
// The creator of a PartialError should include the failed signal data so that the preceding
// components in the pipeline can use this information for partial retries.
type PartialError struct {
	error
	failedTraces  pdata.Traces
	failedLogs    pdata.Logs
	failedMetrics pdata.Metrics
}

// PartialTracesError creates a PartialError for failed traces.
// Use this error type only when a subset of the received data failed to be processed or sent.
func PartialTracesError(err error, failedTraces pdata.Traces) error {
	return PartialError{
		error:        err,
		failedTraces: failedTraces,
	}
}

// GetTraces returns failed traces.
func (err PartialError) GetTraces() pdata.Traces {
	return err.failedTraces
}

// PartialLogsError creates a PartialError for failed logs.
// Use this error type only when a subset of the received data failed to be processed or sent.
func PartialLogsError(err error, failedLogs pdata.Logs) error {
	return PartialError{
		error:      err,
		failedLogs: failedLogs,
	}
}

// GetLogs returns failed logs.
func (err PartialError) GetLogs() pdata.Logs {
	return err.failedLogs
}

// PartialMetricsError creates a PartialError for failed metrics.
// Use this error type only when a subset of the received data failed to be processed or sent.
func PartialMetricsError(err error, failedMetrics pdata.Metrics) error {
	return PartialError{
		error:         err,
		failedMetrics: failedMetrics,
	}
}

// GetMetrics returns failed metrics.
func (err PartialError) GetMetrics() pdata.Metrics {
	return err.failedMetrics
}

// IsPartial checks if an error was wrapped with a PartialError.
func IsPartial(err error) bool {
	if err != nil {
		return errors.As(err, &PartialError{})
	}
	return false
}
