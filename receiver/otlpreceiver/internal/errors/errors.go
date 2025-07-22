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
	if err == nil {
		return nil
	}

	// Handle client disconnection errors first. These are typically transient
	// and should lead to a retryable `codes.Unavailable`.
	if IsClientDisconnectError(err) {
		return status.New(codes.Unavailable, http.StatusText(http.StatusServiceUnavailable)).Err()
	}

	// Handle permanent errors. These indicate a server-side issue that won't resolve
	if consumererror.IsPermanent(err) {
		return status.New(codes.Internal, err.Error()).Err()
	}

	s, ok := status.FromError(err)
	if ok {
		return s.Err()
	}

	return status.New(codes.Unavailable, err.Error()).Err()
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
