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

// TraceError is an error that may carry associated Trace data for a subset of received data
// that faiiled to be processed or sent.
type TraceError struct {
	error
	failed pdata.Traces
}

// Traces creates a TraceError that can encapsulate received data that failed to be processed or sent.
func Traces(err error, failed pdata.Traces) error {
	return TraceError{
		error:  err,
		failed: failed,
	}
}

// IsTrace checks if an error includes a TraceError.
func IsTrace(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &TraceError{})
}

// GetTraces returns failed traces from the provided error.
func GetTraces(err error) pdata.Traces {
	var res pdata.Traces
	if traceError, ok := err.(TraceError); ok {
		res = traceError.failed
	}
	return res
}

// LogError is an error that may carry associated Log data for a subset of received data
// that faiiled to be processed or sent.
type LogError struct {
	error
	failed pdata.Logs
}

// Logs creates a LogError that can encapsulate received data that failed to be processed or sent.
func Logs(err error, failed pdata.Logs) error {
	return LogError{
		error:  err,
		failed: failed,
	}
}

// IsLogs checks if an error includes a LogError.
func IsLogs(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &LogError{})
}

// GetLogs returns failed logs from the provided error.
func GetLogs(err error) pdata.Logs {
	var res pdata.Logs
	if logError, ok := err.(LogError); ok {
		res = logError.failed
	}
	return res
}

// MetricError is an error that may carry associated Metrics data for a subset of received data
// that faiiled to be processed or sent.
type MetricError struct {
	error
	failed pdata.Metrics
}

// Metrics creates a MetricError that can encapsulate received data that failed to be processed or sent.
func Metrics(err error, failed pdata.Metrics) error {
	return MetricError{
		error:  err,
		failed: failed,
	}
}

// IsMetrics checks if an error includes a MetricError.
func IsMetrics(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &MetricError{})
}

// GetMetrics returns failed metrics from the provided error.
func GetMetrics(err error) pdata.Metrics {
	var res pdata.Metrics
	if metricError, ok := err.(MetricError); ok {
		res = metricError.failed
	}
	return res
}

// IsPartial is a convenience for testing whether an error is any of the consumererror types
// that may convey information about partial processing failures.
func IsPartial(err error) bool {
	return IsTrace(err) || IsMetrics(err) || IsLogs(err)
}
