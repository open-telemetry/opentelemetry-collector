// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package errors // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/util"

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_GetStatusFromError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected *status.Status
	}{
		{
			name:     "Status",
			input:    status.Error(codes.Aborted, "test"),
			expected: status.New(codes.Aborted, "test"),
		},
		{
			name:     "Permanent Error",
			input:    consumererror.NewPermanent(errors.New("test")),
			expected: status.New(codes.Internal, "Permanent error: test"),
		},
		{
			name:     "Non-Permanent Error",
			input:    errors.New("test"),
			expected: status.New(codes.Unavailable, "test"),
		},
		{
			name:     "Client Disconnect Error (Canceled)",
			input:    status.Error(codes.Canceled, "client canceled"),
			expected: status.New(codes.Unavailable, "Service Unavailable"),
		},
		{
			name:     "Client Disconnect Error (Unavailable)",
			input:    status.Error(codes.Unavailable, "connection lost"),
			expected: status.New(codes.Unavailable, "Service Unavailable"),
		},
		{
			name:     "Client Disconnect Error (DeadlineExceeded)",
			input:    status.Error(codes.DeadlineExceeded, "context deadline exceeded"),
			expected: status.New(codes.Unavailable, "Service Unavailable"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusFromError(tt.input)
			assert.Equal(t, tt.expected.Err(), result)
		})
	}
}

func Test_GetHTTPStatusCodeFromStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    *status.Status
		expected int
	}{
		{
			name:     "Retryable Status",
			input:    status.New(codes.Unavailable, "test"),
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "Non-retryable Status",
			input:    status.New(codes.Internal, "test"),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "Specifically 429",
			input:    status.New(codes.ResourceExhausted, "test"),
			expected: http.StatusTooManyRequests,
		},
		{
			name:     "Specifically 400",
			input:    status.New(codes.InvalidArgument, "test"),
			expected: http.StatusBadRequest,
		},
		{
			name:     "Specifically 401",
			input:    status.New(codes.Unauthenticated, "test"),
			expected: http.StatusUnauthorized,
		},
		{
			name:     "Specifically 403",
			input:    status.New(codes.PermissionDenied, "test"),
			expected: http.StatusForbidden,
		},
		{
			name:     "Specifically 404",
			input:    status.New(codes.Unimplemented, "test"),
			expected: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHTTPStatusCodeFromStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
