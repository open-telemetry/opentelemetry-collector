// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"fmt"

	"go.opentelemetry.io/collector/internal/statusconversion"
	"google.golang.org/grpc/status"
)

// StatusError contains either an HTTP or gRPC status code
// resulting from a failed request.
// It allows retrieving a type of error code, converting
// between the status codes supported by each transport if necessary.
//
// It should be created with NewHTTPStatus or NewGRPCStatus.
type StatusError struct {
	error
	httpStatus *int
	grpcStatus *status.Status
}

var _ error = &StatusError{}

// Error returns a string specifying the transport and corresponding status code.
func (se *StatusError) Error() string {
	if se.httpStatus != nil {
		return fmt.Sprintf("HTTP Status: %d", *se.httpStatus)
	} else if se.grpcStatus != nil {
		return fmt.Sprintf("gRPC Status: %s", se.grpcStatus.Code().String())
	} else {
		return "no error code set"
	}
}

func (se *StatusError) Unwrap() error {
	return se.error
}

// HTTPStatus returns an HTTP status code either directly
// set by the source or derived from a gRPC status code set
// by the source.
// If no code has been set, the second return value is set
// to `false`.
func (se *StatusError) HTTPStatus() (int, bool) {
	if se.httpStatus != nil {
		return *se.httpStatus, true
	} else if se.grpcStatus != nil {
		return statusconversion.GetHTTPStatusCodeFromStatus(se.grpcStatus), true
	}

	return 0, false
}

// GRPCStatus returns an gRPC status code either directly
// set by the source or derived from an HTTP status code set
// by the source.
// If no code has been set, the second return value is set
// to `false`.
func (se *StatusError) GRPCStatus() (*status.Status, bool) {
	if se.grpcStatus != nil {
		return se.grpcStatus, true
	} else if se.httpStatus != nil {
		return statusconversion.NewStatusFromMsgAndHTTPCode(se.error.Error(), *se.httpStatus), true
	}

	return &status.Status{}, false
}

// NewHTTPStatus wraps an error with a given HTTP status code.
func NewHTTPStatus(err error, code int) error {
	return &StatusError{
		error:      err,
		httpStatus: &code,
	}
}

// NewHTTPStatus wraps an error with a given gRPC status code.
func NewGRPCStatus(err error, status *status.Status) error {
	return &StatusError{
		error:      err,
		grpcStatus: status,
	}
}
