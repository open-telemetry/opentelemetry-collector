// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func gRPCToHTTP(s *status.Status) int {
	switch s.Code() {
	case codes.Aborted:
		return http.StatusInternalServerError
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.Canceled, codes.DataLoss:
		return http.StatusInternalServerError
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.Unimplemented:
		return http.StatusNotFound
	case codes.Unknown:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
