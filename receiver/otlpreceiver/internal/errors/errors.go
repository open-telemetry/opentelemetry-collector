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

// GetClientDisconnectMessage returns a descriptive message for client disconnection errors
func GetClientDisconnectMessage(err error) string {
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.Canceled:
			return "client canceled the request"
		case codes.Unavailable:
			return "client connection lost"
		case codes.DeadlineExceeded:
			return "client request timed out"
		}
	}
	return "client disconnected"
}

func GetStatusFromError(err error) error {
	s, ok := status.FromError(err)
	if !ok {
		// Default to a retryable error
		// https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#failures
		code := codes.Unavailable
		if consumererror.IsPermanent(err) {
			// If an error is permanent but doesn't have an attached gRPC status, assume it is server-side.
			code = codes.Internal
		}
		s = status.New(code, err.Error())
	} else if IsClientDisconnectError(err) {
		// For client disconnection errors, we want to ensure they are marked as retryable
		// and provide a descriptive message
		s = status.New(codes.Unavailable, GetClientDisconnectMessage(err))
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
