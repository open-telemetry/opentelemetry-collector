// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror/internal/statusconversion"
)

// TransportError contains either an HTTP or gRPC status code
// resulting from a failed synchronous network request.
// It allows retrieving a type of error code, converting
// between the status codes supported by each transport if necessary.
//
// It should be created with NewHTTP or NewGRPC.
type TransportError struct {
	err        error
	httpStatus *int
	grpcStatus *status.Status
}

var _ error = TransportError{}

// Error returns a string specifying the transport and corresponding status code.
func (se TransportError) Error() string {
	if se.httpStatus != nil {
		return fmt.Sprintf("HTTP Status (%d): %s", *se.httpStatus, se.err.Error())
	} else if se.grpcStatus != nil {
		return fmt.Sprintf("gRPC Status (%s): %s", se.grpcStatus.Code().String(), se.err.Error())
	}

	return "Network error (no error code set): " + se.err.Error()
}

func (se TransportError) Unwrap() error {
	return se.err
}

// HTTPStatus returns an HTTP status code either directly
// set by the source or derived from a gRPC status code set
// by the source.
//
// If no code has been set, the return value is
// an HTTP 500 code.
func (se TransportError) HTTPStatus() int {
	if se.httpStatus != nil {
		return *se.httpStatus
	} else if se.grpcStatus != nil {
		return statusconversion.GetHTTPStatusCodeFromStatus(se.grpcStatus)
	}

	return http.StatusInternalServerError
}

// GRPCStatus returns an gRPC status code either directly
// set by the source or derived from an HTTP status code set
// by the source.
//
// If no code has been set, the return value is set
// to `false`.
func (se TransportError) GRPCStatus() *status.Status {
	if se.grpcStatus != nil {
		return se.grpcStatus
	} else if se.httpStatus != nil {
		return statusconversion.NewStatusFromMsgAndHTTPCode(se.err.Error(), *se.httpStatus)
	}

	return status.New(codes.Unknown, se.Error())
}

// NewHTTP wraps an error with a given HTTP status code.
func NewHTTP(err error, code int) error {
	return TransportError{
		err:        err,
		httpStatus: &code,
	}
}

// NewGRPC wraps an error with a given gRPC status code.
func NewGRPC(err error, status *status.Status) error {
	return TransportError{
		err:        err,
		grpcStatus: status,
	}
}
