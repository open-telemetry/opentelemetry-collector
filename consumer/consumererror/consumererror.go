// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/grpc/status"
)

type Error struct {
	error
	retryable  bool
	rejected   int
	delay      time.Duration
	hasTraces  bool
	traces     ptrace.Traces
	hasMetrics bool
	metrics    pmetric.Metrics
	hasLogs    bool
	logs       plog.Logs
	httpStatus int
	grpcStatus *status.Status
}

var noDataCount = -1

// ErrorOption allows annotating an Error with metadata.
type ErrorOption func(error *Error)

// NewConsumerError wraps an error that happened while consuming telemetry
// and adds metadata onto it to be passed back up the pipeline.
func NewConsumerError(origErr error, options ...ErrorOption) error {
	err := Error{error: origErr, rejected: noDataCount}

	cErr := Error{}
	if errors.As(err, &cErr) {
		err.copyMetadata(cErr)
	}

	for _, option := range options {
		option(&err)
	}

	return err
}

func (e Error) Error() string {
	return e.error.Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (e Error) Unwrap() error {
	return e.error
}

func (e Error) Rejected() int {
	return e.rejected
}

func (e Error) Delay() time.Duration {
	return e.delay
}

func (e Error) RetryableTraces() (ptrace.Traces, bool) {
	return e.traces, e.hasTraces
}

func (e Error) RetryableMetrics() (pmetric.Metrics, bool) {
	return e.metrics, e.hasMetrics
}

func (e Error) RetryableLogs() (plog.Logs, bool) {
	return e.logs, e.hasLogs
}

func (e Error) ToHTTP() int {
	// todo: translate gRPC to HTTP status
	return e.httpStatus
}

func (e Error) ToGRPC() *status.Status {
	// todo: translate HTTP to grPC status
	return e.grpcStatus
}

// IsPermanent checks if an error was wrapped with the NewPermanent function, which
// is used to indicate that a given error will always be returned in the case
// that its sources receives the same input.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}

	cErr := Error{}
	if errors.As(err, &cErr) {
		return !cErr.retryable
	}

	return false
}

func (e *Error) copyMetadata(err Error) {
	e.rejected = err.rejected
	e.httpStatus = err.httpStatus
	e.grpcStatus = err.grpcStatus
}

func WithRejectedCount(count int) ErrorOption {
	return func(err *Error) {
		err.rejected = count
	}
}

func WithHTTPStatus(status int) ErrorOption {
	return func(err *Error) {
		err.httpStatus = status
	}
}

func WithGRPCStatus(status *status.Status) ErrorOption {
	return func(err *Error) {
		err.grpcStatus = status
	}
}

type RetryOption func(err *Error)

func WithRetryableTraces(td ptrace.Traces, options ...RetryOption) ErrorOption {
	return func(err *Error) {
		err.traces = td
		err.hasTraces = true
		err.retryable = true
		for _, option := range options {
			option(err)
		}
	}
}

func WithRetryableMetrics(md pmetric.Metrics, options ...RetryOption) ErrorOption {
	return func(err *Error) {
		err.metrics = md
		err.hasMetrics = true
		err.retryable = true
		for _, option := range options {
			option(err)
		}
	}
}

func WithRetryableLogs(ld plog.Logs, options ...RetryOption) ErrorOption {
	return func(err *Error) {
		err.logs = ld
		err.hasLogs = true
		err.retryable = true
		for _, option := range options {
			option(err)
		}
	}
}
