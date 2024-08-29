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
// It may hold multiple errors from downstream components, and can
// be merged with other errors as it travels upstream using `Combine`. `Error`
// should be obtained using `errors.As`. If this is not possible, the most
// `ErrorContainer` object should be highest on the stack.
//
// Experimental: This API is at the early stage of development and may change
// without backward compatibility
type Error struct {
	error
	httpStatus *int
	grpcStatus *status.Status

	errors []*Error
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
	e := &Error{errors: make([]*Error, 0, len(errs))}

	resultingStatus := 0

	for _, err := range errs {
		var otherErr *Error
		if errors.As(err, &otherErr) {
			if otherErr.httpStatus != nil {
				resultingStatus = aggregateStatuses(resultingStatus, *otherErr.httpStatus)
			} else if otherErr.grpcStatus != nil {
				resultingStatus = aggregateStatuses(resultingStatus, statusconversion.GetHTTPStatusCodeFromStatus(otherErr.grpcStatus))
			}

			e.errors = append(e.errors, otherErr.errors...)
		} else {
			e.errors = append(e.errors, &Error{error: err})
		}
	}

	if resultingStatus != 0 {
		e.httpStatus = &resultingStatus
	}

	return e
}

// WithHTTPStatus records an HTTP status code that was received from a server
// during data submission.
func WithHTTPStatus(status int) ErrorOption {
	return errorOptionFunc(func(err *Error) {
		err.httpStatus = &status
	})
}

// WithGRPCStatus records a gRPC status code that was received from a server
// during data submission.
func WithGRPCStatus(status *status.Status) ErrorOption {
	return errorOptionFunc(func(err *Error) {
		err.grpcStatus = status
	})
}

// Error implements the error interface.
func (ed *Error) Error() string {
	return ed.error.Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (e *Error) Unwrap() []error {
	errors := make([]error, 0, len(e.errors)+1)

	errors = append(errors, e.error)

	for _, err := range e.errors {
		errors = append(errors, err)
	}

	return errors
}

// HTTPStatus returns an HTTP status code either directly set by the source or
// derived from a gRPC status code set by the source. If both statuses are set,
// the HTTP status code is returned.
//
// If no code has been set, the second return value is `false`.
func (ed *Error) HTTPStatus() (int, bool) {
	if ed.httpStatus != nil {
		return *ed.httpStatus, true
	} else if ed.grpcStatus != nil {
		return statusconversion.GetHTTPStatusCodeFromStatus(ed.grpcStatus), true
	}

	return 0, false
}

// GRPCStatus returns an gRPC status code either directly set by the source or
// derived from an HTTP status code set by the source. If both statuses are set,
// the gRPC status code is returned.
//
// If no code has been set, the second return value is `false`.
func (ed *Error) GRPCStatus() (*status.Status, bool) {
	if ed.grpcStatus != nil {
		return ed.grpcStatus, true
	} else if ed.httpStatus != nil {
		return statusconversion.NewStatusFromMsgAndHTTPCode(ed.Error(), *ed.httpStatus), true
	}

	return nil, false
}

func aggregateStatuses(a int, b int) int {
	switch {
	// If a is unset, keep b
	case a == 0:
		return b
	// If b is unset, keep a
	case b == 0:
		return a
	// If a and b have been set and one is a 4xx code, the correct code is
	// ambiguous, so return a 400, which is permanent.
	case (a >= 400 && a < 500) || (b >= 400 && b < 500):
		return http.StatusBadRequest
	// If a and b have been set and one is a 5xx code, the correct code is
	// ambiguous, so return a 500, which is permanent.
	case a >= 500 || b >= 500:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
