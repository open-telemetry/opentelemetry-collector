// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror/internal/statusconversion"
)

// Error is intended to be used to encapsulate various information that can add
// context to an error that occurred within a pipeline component. Error objects
// are constructed through calling `New` with the relevant options to capture
// data around the error that occurred.
//
// It may hold multiple errors from downstream components, and can be merged
// with other errors as it travels upstream using `Combine`. The `Error` should
// be obtained from a given `error` object using `errors.As`.
//
// Experimental: This API is at the early stage of development and may change
// without backward compatibility
type Error struct {
	error
	httpStatus      int
	grpcStatus      *status.Status
	partialMsg      string
	partialRejected int64
}

var _ error = (*Error)(nil)

// ErrorOption allows annotating an Error with metadata.
type ErrorOption interface {
	applyOption(*Error)
}

type errorOptionFunc func(*Error)

func (f errorOptionFunc) applyOption(e *Error) {
	f(e)
}

// New wraps an error that happened while consuming telemetry and adds metadata
// onto it to be passed back up the pipeline.
//
// Experimental: This API is at the early stage of development and may change
// without backward compatibility
func New(origErr error, options ...ErrorOption) error {
	err := &Error{error: origErr}

	for _, option := range options {
		option.applyOption(err)
	}

	return err
}

// Combine joins errors into a single `Error` object. The component that
// initiated the data submission can then work with the `Error` object to
// understand the failure.
func Combine(errs ...error) error {
	e := &Error{error: errors.Join(errs...)}

	resultingStatus := 0

	for _, err := range errs {
		var otherErr *Error
		if errors.As(err, &otherErr) {
			if otherErr.httpStatus != 0 {
				resultingStatus = aggregateStatuses(resultingStatus, otherErr.httpStatus)
			} else if otherErr.grpcStatus != nil {
				resultingStatus = aggregateStatuses(resultingStatus, statusconversion.GetHTTPStatusCodeFromStatus(otherErr.grpcStatus))
			}
		}
	}

	if resultingStatus != 0 {
		e.httpStatus = resultingStatus
	}

	return e
}

// WithHTTPStatus records an HTTP status code that was received from a server
// during data submission.
func WithHTTPStatus(status int) ErrorOption {
	return errorOptionFunc(func(err *Error) {
		err.httpStatus = status
	})
}

// WithGRPCStatus records a gRPC status code that was received from a server
// during data submission.
func WithGRPCStatus(status *status.Status) ErrorOption {
	return errorOptionFunc(func(err *Error) {
		err.grpcStatus = status
	})
}

// Experimental function for demonstration purposes only
func WithExperimentalPartial(msg string, rejected int64) ErrorOption {
	return errorOptionFunc(func(err *Error) {
		err.partialMsg = msg
		err.partialRejected = rejected
	})
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.error.Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (e *Error) Unwrap() error {
	return e.error
}

// HTTPStatus returns an HTTP status code either directly set by the source or
// derived from a gRPC status code set by the source. If both statuses are set,
// the HTTP status code is returned.
//
// If no code has been set, the second return value is `false`.
func (e *Error) HTTPStatus() (int, bool) {
	if e.httpStatus != 0 {
		return e.httpStatus, true
	} else if e.grpcStatus != nil {
		return statusconversion.GetHTTPStatusCodeFromStatus(e.grpcStatus), true
	}

	return 0, false
}

// GRPCStatus returns an gRPC status code either directly set by the source or
// derived from an HTTP status code set by the source. If both statuses are set,
// the gRPC status code is returned.
//
// If no code has been set, the second return value is `false`.
func (e *Error) GRPCStatus() (*status.Status, bool) {
	if e.grpcStatus != nil {
		return e.grpcStatus, true
	} else if e.httpStatus != 0 {
		return statusconversion.NewStatusFromMsgAndHTTPCode(e.Error(), e.httpStatus), true
	}

	return nil, false
}

// Experimental function for demonstration purposes only
func (e *Error) ExperimentalPartialSuccess() (string, int64, bool) {
	return e.partialMsg, e.partialRejected, e.partialMsg != "" || e.partialRejected != 0
}

func aggregateStatuses(a int, b int) int {
	switch {
	// If a is unset, keep b. b is guaranteed to be non-zero by the caller.
	case a == 0:
		return b
	// If a and b have been set and one is a 5xx code, the correct code is
	// ambiguous, so return a 500, which is permanent.
	case a >= 500 || b >= 500:
		return http.StatusInternalServerError
	// If a and b have been set and one is a 4xx code, the correct code is
	// ambiguous, so return a 400, which is permanent.
	case (a >= 400 && a < 500) || (b >= 400 && b < 500):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
