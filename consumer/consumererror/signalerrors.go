// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"time"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type retryableCommon struct {
	error
	delay time.Duration
}

type retryable[V ptrace.Traces | pmetric.Metrics | plog.Logs] struct {
	retryableCommon
	data V
}

var _ error = retryable[ptrace.Traces]{}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (err retryable[V]) Unwrap() error {
	return err.error
}

// Data returns the telemetry data that failed to be processed or sent.
func (err retryable[V]) Data() V {
	return err.data
}

// Delay returns the time duration before the telemetry data should
// be resent to the destination.
func (err retryable[V]) Delay() time.Duration {
	return err.delay
}

// Traces is an error that may carry associated Trace data for a subset of received data
// that failed to be processed or sent.
type Traces struct {
	retryable[ptrace.Traces]
}

// NewTraces creates a Traces that can encapsulate received data that failed to be processed or sent.
func NewTraces(err error, data ptrace.Traces) error {
	t := Traces{
		retryable: retryable[ptrace.Traces]{
			retryableCommon: retryableCommon{
				error: err,
			},
			data: data,
		},
	}

	return t
}

// Logs is an error that may carry associated Log data for a subset of received data
// that failed to be processed or sent.
type Logs struct {
	retryable[plog.Logs]
}

// NewLogs creates a Logs that can encapsulate received data that failed to be processed or sent.
func NewLogs(err error, data plog.Logs) error {
	l := Logs{
		retryable: retryable[plog.Logs]{
			retryableCommon: retryableCommon{
				error: err,
			},
			data: data,
		},
	}

	return l
}

// Metrics is an error that may carry associated Metrics data for a subset of received data
// that failed to be processed or sent.
type Metrics struct {
	retryable[pmetric.Metrics]
}

// NewMetrics creates a Metrics that can encapsulate received data that failed to be processed or sent.
func NewMetrics(err error, data pmetric.Metrics) error {
	m := Metrics{
		retryable: retryable[pmetric.Metrics]{
			retryableCommon: retryableCommon{
				error: err,
			},
			data: data,
		},
	}

	return m
}
