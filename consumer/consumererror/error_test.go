// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errTest = errors.New("consumererror testing error")

func Test_NewOTLPHTTPError(t *testing.T) {
	httpStatus := 500
	wantErr := &Error{
		error:      errTest,
		httpStatus: httpStatus,
	}

	newErr := NewOTLPHTTPError(errTest, httpStatus)

	require.Equal(t, wantErr, newErr)
}

func Test_NewOTLPGRPCError(t *testing.T) {
	grpcStatus := status.New(codes.Aborted, "aborted")
	wantErr := &Error{
		error:      errTest,
		grpcStatus: grpcStatus,
	}

	newErr := NewOTLPGRPCError(errTest, grpcStatus)

	require.Equal(t, wantErr, newErr)
}

func Test_NewRetryableError(t *testing.T) {
	wantErr := &Error{
		error:     errTest,
		retryable: true,
	}

	newErr := NewRetryableError(errTest)

	require.Equal(t, wantErr, newErr)
}

func Test_Error(t *testing.T) {
	newErr := Error{error: errTest}

	require.Equal(t, errTest.Error(), newErr.Error())
}

func TestUnwrap(t *testing.T) {
	err := &Error{
		error: errTest,
	}

	unwrapped := err.Unwrap()

	require.Equal(t, errTest, unwrapped)
}

func TestAs(t *testing.T) {
	err := &Error{
		error: errTest,
	}

	secondError := errors.Join(errors.New("test"), err)

	var e *Error
	require.ErrorAs(t, secondError, &e)
	assert.Equal(t, errTest.Error(), e.Error())
}

func TestError_Error(t *testing.T) {
	err := &Error{
		error: errTest,
	}

	require.Equal(t, errTest.Error(), err.Error())
}

func TestError_Unwrap(t *testing.T) {
	err := &Error{
		error: errTest,
	}

	require.Equal(t, errTest, err.Unwrap())
}

func TestError_OTLPHTTPStatus(t *testing.T) {
	serverErr := http.StatusTooManyRequests
	testCases := []struct {
		name       string
		httpStatus int
		grpcStatus *status.Status
		want       int
		hasCode    bool
	}{
		{
			name:       "Passes through HTTP status",
			httpStatus: serverErr,
			want:       serverErr,
			hasCode:    true,
		},
		{
			name:       "Converts gRPC status",
			grpcStatus: status.New(codes.ResourceExhausted, errTest.Error()),
			want:       serverErr,
			hasCode:    true,
		},
		{
			name:       "Passes through HTTP status when gRPC status also present",
			httpStatus: serverErr,
			grpcStatus: status.New(codes.OK, errTest.Error()),
			want:       serverErr,
			hasCode:    true,
		},
		{
			name: "No statuses set",
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := Error{
				error:      errTest,
				httpStatus: tt.httpStatus,
				grpcStatus: tt.grpcStatus,
			}

			s := err.OTLPHTTPStatus()

			require.Equal(t, tt.want, s)
		})
	}
}

func TestError_OTLPGRPCStatus(t *testing.T) {
	httpStatus := http.StatusTooManyRequests
	otherOTLPHTTPStatus := http.StatusOK
	serverErr := status.New(codes.ResourceExhausted, errTest.Error())
	testCases := []struct {
		name       string
		httpStatus int
		grpcStatus *status.Status
		want       *status.Status
		hasCode    bool
	}{
		{
			name:       "Converts HTTP status",
			httpStatus: httpStatus,
			want:       serverErr,
			hasCode:    true,
		},
		{
			name:       "Passes through gRPC status",
			grpcStatus: serverErr,
			want:       serverErr,
			hasCode:    true,
		},
		{
			name:       "Passes through gRPC status when gRPC status also present",
			httpStatus: otherOTLPHTTPStatus,
			grpcStatus: serverErr,
			want:       serverErr,
			hasCode:    true,
		},
		{
			name: "No statuses set",
			want: status.New(codes.Internal, errTest.Error()),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := Error{
				error:      errTest,
				httpStatus: tt.httpStatus,
				grpcStatus: tt.grpcStatus,
			}

			s := err.OTLPGRPCStatus()

			require.Equal(t, tt.want, s)
		})
	}
}
