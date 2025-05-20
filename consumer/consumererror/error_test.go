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
		retryable  bool
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
			name:       "Passes through HTTP status when retryable also present",
			httpStatus: serverErr,
			retryable:  true,
			want:       serverErr,
		},
		{
			name:      "No statuses set with retryable",
			retryable: true,
			want:      http.StatusServiceUnavailable,
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
				retryable:  tt.retryable,
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
		retryable  bool
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
			name:       "Passes through gRPC status when HTTP status also present",
			httpStatus: otherOTLPHTTPStatus,
			grpcStatus: serverErr,
			want:       serverErr,
			hasCode:    true,
		},
		{
			name:       "Passes through gRPC status when retryable also present",
			grpcStatus: serverErr,
			retryable:  true,
			want:       serverErr,
		},
		{
			name:      "No statuses set with retryable",
			retryable: true,
			want:      status.New(codes.Unavailable, errTest.Error()),
		},
		{
			name: "No statuses set",
			want: status.New(codes.Unknown, errTest.Error()),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := Error{
				error:      errTest,
				httpStatus: tt.httpStatus,
				grpcStatus: tt.grpcStatus,
				retryable:  tt.retryable,
			}

			s := err.OTLPGRPCStatus()

			require.Equal(t, tt.want, s)
		})
	}
}

func TestError_Retryable(t *testing.T) {
	retryableCodes := []codes.Code{codes.Canceled, codes.DeadlineExceeded, codes.Aborted, codes.OutOfRange, codes.Unavailable, codes.DataLoss}
	retryableStatuses := []*status.Status{}

	for _, code := range retryableCodes {
		retryableStatuses = append(retryableStatuses, status.New(code, errTest.Error()))
	}

	nonretryableCodes := []codes.Code{codes.Unauthenticated, codes.Unknown, codes.NotFound, codes.InvalidArgument}
	nonretryableStatuses := []*status.Status{}

	for _, code := range nonretryableCodes {
		nonretryableStatuses = append(nonretryableStatuses, status.New(code, errTest.Error()))
	}

	testCases := []struct {
		name         string
		httpStatuses []int
		grpcStatuses []*status.Status
		retryable    bool
		want         bool
	}{
		{
			name:         "HTTP statuses: retryable",
			httpStatuses: []int{http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout},
			grpcStatuses: []*status.Status{nil},
			want:         true,
		},
		{
			name:         "HTTP statuses: non-retryable",
			httpStatuses: []int{0, http.StatusInternalServerError, http.StatusNotFound, http.StatusUnauthorized},
			grpcStatuses: []*status.Status{nil},
			want:         false,
		},
		{
			name:         "gRPC statuses: retryable",
			httpStatuses: []int{0},
			grpcStatuses: retryableStatuses,
			want:         true,
		},
		{
			name:         "gRPC statuses: non-retryable",
			httpStatuses: []int{0},
			grpcStatuses: nonretryableStatuses,
			want:         false,
		},
		{
			name:         "Retryable set to true",
			httpStatuses: []int{0},
			grpcStatuses: []*status.Status{nil},
			retryable:    true,
			want:         true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			for _, httpStatus := range tt.httpStatuses {
				for _, grpcStatus := range tt.grpcStatuses {
					err := Error{
						error:      errTest,
						httpStatus: httpStatus,
						grpcStatus: grpcStatus,
						retryable:  tt.retryable,
					}

					isRetryable := err.Retryable()

					require.Equal(t, tt.want, isRetryable)
				}
			}
		})
	}
}
