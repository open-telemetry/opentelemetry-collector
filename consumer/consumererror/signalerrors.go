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

// Traces is an error that may carry associated Trace data for a subset of received data
// that faiiled to be processed or sent.
type Traces struct {
	error
	failed pdata.Traces
}

// NewTraces creates a Traces that can encapsulate received data that failed to be processed or sent.
func NewTraces(err error, failed pdata.Traces) error {
	return Traces{
		error:  err,
		failed: failed,
	}
}

// IsTraces checks if an error includes a Traces.
func IsTraces(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &Traces{})
}

// GetTraces returns failed traces from the provided error.
func GetTraces(err error) pdata.Traces {
	var res pdata.Traces
	if traceError, ok := err.(Traces); ok {
		res = traceError.failed
	}
	return res
}

// Logs is an error that may carry associated Log data for a subset of received data
// that faiiled to be processed or sent.
type Logs struct {
	error
	failed pdata.Logs
}

// NewLogs creates a Logs that can encapsulate received data that failed to be processed or sent.
func NewLogs(err error, failed pdata.Logs) error {
	return Logs{
		error:  err,
		failed: failed,
	}
}

// IsLogs checks if an error includes a Logs.
func IsLogs(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &Logs{})
}

// GetLogs returns failed logs from the provided error.
func GetLogs(err error) pdata.Logs {
	var res pdata.Logs
	if logError, ok := err.(Logs); ok {
		res = logError.failed
	}
	return res
}

// Metrics is an error that may carry associated Metrics data for a subset of received data
// that faiiled to be processed or sent.
type Metrics struct {
	error
	failed pdata.Metrics
}

// NewMetrics creates a Metrics that can encapsulate received data that failed to be processed or sent.
func NewMetrics(err error, failed pdata.Metrics) error {
	return Metrics{
		error:  err,
		failed: failed,
	}
}

// IsMetrics checks if an error includes a Metrics.
func IsMetrics(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &Metrics{})
}

// GetMetrics returns failed metrics from the provided error.
func GetMetrics(err error) pdata.Metrics {
	var res pdata.Metrics
	if metricError, ok := err.(Metrics); ok {
		res = metricError.failed
	}
	return res
}

// IsPartial is a convenience for testing whether an error is any of the consumererror types
// that may convey information about partial processing failures.
func IsPartial(err error) bool {
	return IsTraces(err) || IsMetrics(err) || IsLogs(err)
}
