// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statusconversion // import "go.opentelemetry.io/collector/consumer/consumererror/internal/statusconversion"

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
			input:    status.New(codes.InvalidArgument, "test"),
			expected: http.StatusBadRequest,
		},
		{
			name:     "Specifically 429",
			input:    status.New(codes.ResourceExhausted, "test"),
			expected: http.StatusTooManyRequests,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHTTPStatusCodeFromStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_ErrorMsgAndHTTPCodeToStatus(t *testing.T) {
	tests := []struct {
		name       string
		errMsg     string
		statusCode int
		expected   *status.Status
	}{
		{
			name:       "Bad Request",
			errMsg:     "test",
			statusCode: http.StatusBadRequest,
			expected:   status.New(codes.InvalidArgument, "test"),
		},
		{
			name:       "Unauthorized",
			errMsg:     "test",
			statusCode: http.StatusUnauthorized,
			expected:   status.New(codes.Unauthenticated, "test"),
		},
		{
			name:       "Forbidden",
			errMsg:     "test",
			statusCode: http.StatusForbidden,
			expected:   status.New(codes.PermissionDenied, "test"),
		},
		{
			name:       "Not Found",
			errMsg:     "test",
			statusCode: http.StatusNotFound,
			expected:   status.New(codes.Unimplemented, "test"),
		},
		{
			name:       "Too Many Requests",
			errMsg:     "test",
			statusCode: http.StatusTooManyRequests,
			expected:   status.New(codes.ResourceExhausted, "test"),
		},
		{
			name:       "Bad Gateway",
			errMsg:     "test",
			statusCode: http.StatusBadGateway,
			expected:   status.New(codes.Unavailable, "test"),
		},
		{
			name:       "Service Unavailable",
			errMsg:     "test",
			statusCode: http.StatusServiceUnavailable,
			expected:   status.New(codes.Unavailable, "test"),
		},
		{
			name:       "Gateway Timeout",
			errMsg:     "test",
			statusCode: http.StatusGatewayTimeout,
			expected:   status.New(codes.Unavailable, "test"),
		},
		{
			name:       "Unsupported Media Type",
			errMsg:     "test",
			statusCode: http.StatusUnsupportedMediaType,
			expected:   status.New(codes.Unknown, "test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewStatusFromMsgAndHTTPCode(tt.errMsg, tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}
