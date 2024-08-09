// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"

	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror/internal/statusconversion"
)

// ErrorContainer acts as a container for errors coming from pipeline components.
// It may hold multiple errors from downstream components, and is designed
// to act as a way to accumulate errors as it travels upstream in a pipeline.
// `ErrorContainer` should be obtained using `errors.As` and as a result, ideally
// a single instance should exist in an error stack. If this is not possible,
// the most `ErrorContainer` object should be highest on the stack.
//
// Experimental: This API is at the early stage of development and may change without backward compatibility
type ErrorContainer struct {
	errors []Error
}

// Error is intended to be used to encapsulate various information
// that can add context to an error that occurred within a pipeline component.
// Error objects are constructed through calling `New` with the relevant
// options to capture data around the error that occurred. It can then be pulled
// out by an upstream component by calling `Error.Data`.
//
// Experimental: This API is at the early stage of development and may change without backward compatibility
type Error interface {
	Error() string

	Unwrap() error

	// Returns a status code and a boolean indicating whether a status was set
	// by the error creator.
	HTTPStatus() (int, bool)

	// Returns a status code and a boolean indicating whether a status was set
	// by the error creator.
	GRPCStatus() (*status.Status, bool)

	// Disallow implementations outside this package to allow later extending
	// the interface.
	unexported()
}

type errorData struct {
	error
	httpStatus *int
	grpcStatus *status.Status
}

// ErrorOption allows annotating an Error with metadata.
type ErrorOption interface {
	applyOption(*errorData)
}

type errorOptionFunc func(*errorData)

func (f errorOptionFunc) applyOption(e *errorData) {
	f(e)
}

// New wraps an error that happened while consuming telemetry
// and adds metadata onto it to be passed back up the pipeline.
//
// Experimental: This API is at the early stage of development and may change without backward compatibility
func New(origErr error, options ...ErrorOption) error {
	errData := &errorData{error: origErr}
	err := &ErrorContainer{errors: []Error{errData}}

	for _, option := range options {
		option.applyOption(errData)
	}

	return err
}

var _ error = (*ErrorContainer)(nil)

// Error implements the `error` interface.
func (e *ErrorContainer) Error() string {
	return e.errors[len(e.errors)-1].Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (e *ErrorContainer) Unwrap() []error {
	errors := make([]error, 0, len(e.errors))

	for _, err := range e.errors {
		errors = append(errors, err)
	}

	return errors
}

// Errors returns all the accumulated Error objects
// emitted by components downstream in the pipeline.
// These can then be aggregated or worked with individually.
func (e *ErrorContainer) Errors() []Error {
	return e.errors
}

// Combine joins errors that occur at a fanout into a single
// `Error` object. The component that created the data submission
// can then work with the `Error` object to understand the failure.
func Combine(errs ...error) *ErrorContainer {
	e := &ErrorContainer{errors: make([]Error, 0, len(errs))}

	for _, err := range errs {
		var otherErr *ErrorContainer
		if errors.As(err, &otherErr) {
			e.errors = append(e.errors, otherErr.errors...)
		} else {
			e.errors = append(e.errors, &errorData{error: err})
		}
	}

	return e
}

// WithHTTPStatus records an HTTP status code that was received
// from a server during data submission.
func WithHTTPStatus(status int) ErrorOption {
	return errorOptionFunc(func(err *errorData) {
		err.httpStatus = &status
	})
}

// WithGRPCStatus records a gRPC status code that was received
// from a server during data submission.
func WithGRPCStatus(status *status.Status) ErrorOption {
	return errorOptionFunc(func(err *errorData) {
		err.grpcStatus = status
	})
}

var _ error = (*errorData)(nil)

func (ed *errorData) unexported() {}

// Error implements the error interface.
func (ed *errorData) Error() string {
	return ed.error.Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (ed *errorData) Unwrap() error {
	return ed.error
}

// HTTPStatus returns an HTTP status code either directly
// set by the source or derived from a gRPC status code set
// by the source. If both statuses are set, the HTTP status
// code is returned.
//
// If no code has been set, the second return value is `false`.
func (ed *errorData) HTTPStatus() (int, bool) {
	if ed.httpStatus != nil {
		return *ed.httpStatus, true
	} else if ed.grpcStatus != nil {
		return statusconversion.GetHTTPStatusCodeFromStatus(ed.grpcStatus), true
	}

	return 0, false
}

// GRPCStatus returns an gRPC status code either directly
// set by the source or derived from an HTTP status code set
// by the source. If both statuses are set, the gRPC status
// code is returned.
//
// If no code has been set, the second return value is `false`.
func (ed *errorData) GRPCStatus() (*status.Status, bool) {
	if ed.grpcStatus != nil {
		return ed.grpcStatus, true
	} else if ed.httpStatus != nil {
		return statusconversion.NewStatusFromMsgAndHTTPCode(ed.Error(), *ed.httpStatus), true
	}

	return nil, false
}
