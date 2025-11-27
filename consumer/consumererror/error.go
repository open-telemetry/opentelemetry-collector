// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror/internal/statusconversion"
)

// Error is intended to be used to encapsulate various information that can add
// context to an error that occurred within a pipeline component. Error objects
// are constructed through calling `New` with the relevant options to capture
// data around the error that occurred.
//
// Error should be obtained from a given `error` object using `errors.As`.
type Error struct {
	error
	httpStatus  int
	grpcStatus  *status.Status
	isRetryable bool
}

var _ error = (*Error)(nil)

// NewOTLPHTTPError records an HTTP status code that was received from a server
// during data submission.
//
// NOTE: This function will panic if passed an HTTP status between 200 and 299 inclusive.
// This is to reserve the behavior for handling these codes for the future.
func NewOTLPHTTPError(origErr error, httpStatus int) error {
	// Matches what is considered a successful response in the OTLP/HTTP Exporter.
	if httpStatus >= 200 && httpStatus <= 299 {
		panic("NewOTLPHTTPError should not be called with a success code")
	}
	var retryable bool
	switch httpStatus {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		retryable = true
	}

	return &Error{error: origErr, httpStatus: httpStatus, isRetryable: retryable}
}

// NewOTLPGRPCError records a gRPC status code that was received from a server
// during data submission.
//
// NOTE: This function will panic if passed a *status.Status with an underlying
// code of codes.OK. This is to reserve the behavior for handling this code for
// the future.
func NewOTLPGRPCError(origErr error, status *status.Status) error {
	var retryable bool
	if status != nil {
		switch status.Code() {
		case codes.Canceled, codes.DeadlineExceeded, codes.Aborted, codes.OutOfRange, codes.Unavailable, codes.DataLoss:
			retryable = true
		// Matches what is considered a successful response in the OTLP Exporter.
		case codes.OK:
			panic("NewOTLPGRPCError should not be called with an OK code")
		}
	}

	return &Error{error: origErr, grpcStatus: status, isRetryable: retryable}
}

// NewRetryableError records that this error is retryable according to OTLP specification.
func NewRetryableError(origErr error) error {
	return &Error{error: origErr, isRetryable: true}
}

// Error implements the error interface.
//
// If an error object was given, that is used.
// Otherwise, the gRPC error from the status.Status is used,
// or an error message containing the HTTP status code is given.
func (e *Error) Error() string {
	if e.error != nil {
		return e.error.Error()
	}

	if e.grpcStatus != nil {
		return e.grpcStatus.Err().Error()
	} else if e.httpStatus > 0 {
		return fmt.Sprintf("network error: received HTTP status %d", e.httpStatus)
	}

	return "uninitialized consumererror.Error"
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
//
// If an error object was not passed but a gRPC `status.Status` was passed,
// the underlying error from the status is returned.
func (e *Error) Unwrap() error {
	if e.error != nil {
		return e.error
	}

	if e.grpcStatus != nil {
		return e.grpcStatus.Err()
	}

	return nil
}

// IsRetryable returns true if the error was created with NewRetryableError, if
// the HTTP status code is retryable according to OTLP, or if the gRPC status is
// retryable according to OTLP. Otherwise, returns false.
//
// See https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md for retryable
// HTTP and gRPC codes.
func (e *Error) IsRetryable() bool {
	return e.isRetryable
}

// ToHTTPStatus returns an HTTP status code either directly set by the source on
// an [Error] object, derived from a gRPC status code set by the source, or
// derived from Retryable. When deriving the value, the OTLP specification is
// used to map to HTTP. See
// https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md
// for more details.
//
// If a http status code cannot be derived from these three sources then 500 is
// returned.
func ToHTTPStatus(err error) int {
	var e *Error
	if errors.As(err, &e) {
		if e.httpStatus != 0 {
			return e.httpStatus
		}
		if e.grpcStatus != nil {
			return statusconversion.GetHTTPStatusCodeFromStatus(e.grpcStatus)
		}
		if e.isRetryable {
			return http.StatusServiceUnavailable
		}
	}
	return http.StatusInternalServerError
}

// ToGRPCStatus returns a gRPC status code either directly set by the source on
// an [Error] object, derived from an HTTP status code set by the
// source, or derived from Retryable. When deriving the value, the OTLP
// specification is used to map to gRPC. See
// https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md
// for more details.
//
// If an [Error] object is not present, then we attempt to get a status.Status from the
// error tree.
//
// If a status.Status cannot be derived from these sources then INTERNAL is
// returned.
func ToGRPCStatus(err error) *status.Status {
	var e *Error
	if errors.As(err, &e) {
		if e.grpcStatus != nil {
			return e.grpcStatus
		}
		if e.httpStatus != 0 {
			return statusconversion.NewStatusFromMsgAndHTTPCode(e.Error(), e.httpStatus)
		}
		if e.isRetryable {
			return status.New(codes.Unavailable, e.Error())
		}
	}
	if st, ok := status.FromError(err); ok {
		return st
	}
	return status.New(codes.Unknown, e.Error())
}
