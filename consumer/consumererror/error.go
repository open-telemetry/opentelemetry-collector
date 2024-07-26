// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"go.opentelemetry.io/collector/consumer/consumererror/internal/statusconversion"
	"google.golang.org/grpc/status"
)

type Error struct {
	errors []ErrorData
}

type ErrorData interface {
	Error() string

	Unwrap() error

	// Returns nil if a status is not available.
	HTTPStatus() (int, bool)

	// Returns nil if a status is not available.
	GRPCStatus() (*status.Status, bool)
}

type errorData struct {
	error
	httpStatus  *int
	grpcStatus  *status.Status
	extraErrors []error
}

// ErrorOption allows annotating an Error with metadata.
type ErrorOption func(error *errorData)

// NewConsumerError wraps an error that happened while consuming telemetry
// and adds metadata onto it to be passed back up the pipeline.
func NewConsumerError(origErr error, options ...ErrorOption) error {
	err := &Error{}
	errData := &errorData{error: origErr}

	for _, option := range options {
		option(errData)
	}

	return err
}

var _ error = &Error{}

func (e *Error) Error() string {
	return e.errors[len(e.errors)-1].Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (e *Error) Unwrap() []error {
	errors := make([]error, 0, len(e.errors))

	for _, err := range e.errors {
		errors = append(errors, err)
	}

	return errors
}

func (e *Error) Combine(errs ...error) {
	for _, err := range errs {
		if ne, ok := err.(*Error); ok {
			e.errors = append(e.errors, ne.errors...)
		} else {
			e.errors = append(e.errors, &errorData{error: err})
		}
	}
}

func (e *Error) Data() []ErrorData {
	return e.errors
}

func WithHTTPStatus(status int) ErrorOption {
	return func(err *errorData) {
		err.httpStatus = &status
	}
}

func WithGRPCStatus(status *status.Status) ErrorOption {
	return func(err *errorData) {
		err.grpcStatus = status
	}
}

func WithMetadata(err error) ErrorOption {
	return func(errData *errorData) {
		errData.extraErrors = append(errData.extraErrors, err)
	}
}

var _ error = &errorData{}

func (e *errorData) Error() string {
	return e.error.Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (e *errorData) Unwrap() error {
	return e.error
}

// HTTPStatus returns an HTTP status code either directly
// set by the source or derived from a gRPC status code set
// by the source.
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
// by the source.
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
