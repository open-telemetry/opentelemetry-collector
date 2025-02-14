// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statusutil

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

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

func TestGetRetryInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    *status.Status
		expected *errdetails.RetryInfo
	}{
		{
			name:     "NoDetails",
			input:    status.New(codes.InvalidArgument, "test"),
			expected: nil,
		},
		{
			name: "WithRetryInfoDetails",
			input: func() *status.Status {
				st := status.New(codes.ResourceExhausted, "test")
				dt, err := st.WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(1 * time.Second)})
				require.NoError(t, err)
				return dt
			}(),
			expected: &errdetails.RetryInfo{RetryDelay: durationpb.New(1 * time.Second)},
		},
		{
			name: "WithOtherDetails",
			input: func() *status.Status {
				st := status.New(codes.ResourceExhausted, "test")
				dt, err := st.WithDetails(&errdetails.ErrorInfo{Reason: "my reason"})
				require.NoError(t, err)
				return dt
			}(),
			expected: nil,
		},
		{
			name: "WithMultipleDetails",
			input: func() *status.Status {
				st := status.New(codes.ResourceExhausted, "test")
				dt, err := st.WithDetails(
					&errdetails.ErrorInfo{Reason: "my reason"},
					&errdetails.RetryInfo{RetryDelay: durationpb.New(1 * time.Second)})
				require.NoError(t, err)
				return dt
			}(),
			expected: &errdetails.RetryInfo{RetryDelay: durationpb.New(1 * time.Second)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRetryInfo(tt.input)
			assert.True(t, proto.Equal(tt.expected, result))
		})
	}
}
