// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package errors // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

// IsClientDisconnectError returns true if the error indicates a client disconnection
func IsClientDisconnectError(err error) bool {
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.Canceled, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}
	return false
}

func GetStatusFromError(err error) error {
	var s *status.Status
	_, ok := status.FromError(err)

	switch {
	case !ok:
		// Default to a retryable error
		code := codes.Unavailable
		if consumererror.IsPermanent(err) {
			// If an error is permanent but doesn't have an attached gRPC status, assume it is server-side.
			code = codes.Internal
		}
		s = status.New(code, err.Error())

	case IsClientDisconnectError(err):
		// For client disconnection errors, mark as retryable
		code := codes.Unavailable
		httpCode := GetHTTPStatusCodeFromStatus(status.New(code, ""))
		s = status.New(code, http.StatusText(httpCode))

	case consumererror.IsPermanent(err):
		// For permanent errors, treat as client error
		code := codes.InvalidArgument
		httpCode := GetHTTPStatusCodeFromStatus(status.New(code, ""))
		s = status.New(codes.InvalidArgument, http.StatusText(httpCode))

	default:
		// For non-permanent errors, treat as retryable server error
		code := codes.Unavailable
		httpCode := GetHTTPStatusCodeFromStatus(status.New(code, ""))
		s = status.New(code, http.StatusText(httpCode))
	}
	return s.Err()
}

func GetHTTPStatusCodeFromStatus(s *status.Status) int {
	// See https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#failures
	// to see if a code is retryable.
	// See https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#failures-1
	// to see a list of retryable http status codes.
	switch s.Code() {
	// Retryable
	case codes.Canceled, codes.DeadlineExceeded, codes.Aborted, codes.OutOfRange, codes.Unavailable, codes.DataLoss:
		return http.StatusServiceUnavailable
	// Retryable
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	// Not Retryable
	case codes.InvalidArgument:
		return http.StatusBadRequest
	// Not Retryable
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	// Not Retryable
	case codes.PermissionDenied:
		return http.StatusForbidden
	// Not Retryable
	case codes.Unimplemented:
		return http.StatusNotFound
	// Not Retryable
	default:
		return http.StatusInternalServerError
	}
}
