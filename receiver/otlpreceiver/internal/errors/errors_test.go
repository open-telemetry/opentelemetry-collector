// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package errors // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/util"

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer/consumererror"
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

func Test_IsClientDisconnectError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected bool
	}{
		{
			name:     "Canceled",
			input:    status.Error(codes.Canceled, "client canceled"),
			expected: true,
		},
		{
			name:     "Unavailable",
			input:    status.Error(codes.Unavailable, "connection lost"),
			expected: true,
		},
		{
			name:     "DeadlineExceeded",
			input:    status.Error(codes.DeadlineExceeded, "context deadline exceeded"),
			expected: true,
		},
		{
			name:     "OtherError",
			input:    status.Error(codes.Internal, "internal error"),
			expected: false,
		},
		{
			name:     "NonStatusError",
			input:    errors.New("some error"),
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsClientDisconnectError(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_GetClientDisconnectMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected string
	}{
		{
			name:     "Canceled",
			input:    status.Error(codes.Canceled, "client canceled"),
			expected: "client canceled the request",
		},
		{
			name:     "Unavailable",
			input:    status.Error(codes.Unavailable, "connection lost"),
			expected: "client connection lost",
		},
		{
			name:     "DeadlineExceeded",
			input:    status.Error(codes.DeadlineExceeded, "context deadline exceeded"),
			expected: "client request timed out",
		},
		{
			name:     "OtherError",
			input:    status.Error(codes.Internal, "internal error"),
			expected: "client disconnected",
		},
		{
			name:     "NonStatusError",
			input:    errors.New("some error"),
			expected: "client disconnected",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetClientDisconnectMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
