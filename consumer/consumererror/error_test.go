// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errTest = errors.New("consumererror testing error")

func Test_New(t *testing.T) {
	httpStatus := 500
	grpcStatus := status.New(codes.Aborted, "aborted")
	wantErr := &Error{
		error:      errTest,
		httpStatus: httpStatus,
		grpcStatus: grpcStatus,
	}

	newErr := New(errTest,
		WithHTTPStatus(httpStatus),
		WithGRPCStatus(grpcStatus),
	)

	require.Equal(t, wantErr, newErr)
}

func Test_Error(t *testing.T) {
	newErr := New(errTest)

	require.Equal(t, errTest.Error(), newErr.Error())
}

func TestUnwrap(t *testing.T) {
	err := &Error{
		error: errTest,
	}

	unwrapped := err.Unwrap()

	require.Equal(t, errTest, unwrapped)
}

func TestCombine(t *testing.T) {
	err0 := &Error{}
	err1 := &Error{}
	want := &Error{error: errors.Join(err0, err1, errTest)}

	err := Combine(err0, err1, errTest)

	require.Equal(t, want, err)

}

func TestCombineStatusAggregation(t *testing.T) {
	cases := []struct {
		name     string
		errors   []error
		wantHTTP int
		wantGRPC codes.Code
	}{
		{
			name:   "No status codes",
			errors: []error{},
		},
		{
			name: "One status code first",
			errors: []error{
				New(errTest, WithHTTPStatus(http.StatusTooManyRequests)),
				New(errTest),
			},
			wantHTTP: http.StatusTooManyRequests,
			wantGRPC: codes.ResourceExhausted,
		},
		{
			name: "One status code second",
			errors: []error{
				New(errTest),
				New(errTest, WithHTTPStatus(http.StatusTooManyRequests)),
			},
			wantHTTP: http.StatusTooManyRequests,
			wantGRPC: codes.ResourceExhausted,
		},
		{
			name: "Two HTTP statuses",
			errors: []error{
				New(errTest, WithHTTPStatus(http.StatusTooManyRequests)),
				New(errTest, WithHTTPStatus(http.StatusNotFound)),
			},
			wantHTTP: http.StatusBadRequest,
			wantGRPC: codes.InvalidArgument,
		},
		{
			name: "Three HTTP statuses",
			errors: []error{
				New(errTest, WithHTTPStatus(http.StatusTooManyRequests)),
				New(errTest, WithHTTPStatus(http.StatusNotFound)),
				New(errTest, WithHTTPStatus(http.StatusUnauthorized)),
			},
			wantHTTP: http.StatusBadRequest,
			wantGRPC: codes.InvalidArgument,
		},
		{
			name: "Two gRPC statuses",
			errors: []error{
				New(errTest, WithGRPCStatus(status.New(codes.PermissionDenied, ""))),
				New(errTest, WithGRPCStatus(status.New(codes.Unauthenticated, ""))),
			},
			wantHTTP: http.StatusBadRequest,
			wantGRPC: codes.InvalidArgument,
		},
		{
			name: "One HTTP, one gRPC status",
			errors: []error{
				New(errTest, WithHTTPStatus(http.StatusTooManyRequests)),
				New(errTest, WithGRPCStatus(status.New(codes.PermissionDenied, ""))),
			},
			wantHTTP: http.StatusBadRequest,
			wantGRPC: codes.InvalidArgument,
		},
		{
			name: "Two 4xx",
			errors: []error{
				New(errTest, WithHTTPStatus(http.StatusTooManyRequests)),
				New(errTest, WithHTTPStatus(http.StatusUnauthorized)),
			},
			wantHTTP: http.StatusBadRequest,
			wantGRPC: codes.InvalidArgument,
		},
		{
			name: "One 4xx, one 5xx",
			errors: []error{
				New(errTest, WithHTTPStatus(http.StatusTooManyRequests)),
				New(errTest, WithHTTPStatus(http.StatusServiceUnavailable)),
			},
			wantHTTP: http.StatusInternalServerError,
			wantGRPC: codes.Unknown,
		},
		{
			name: "Two 5xx",
			errors: []error{
				New(errTest, WithGRPCStatus(status.New(codes.DeadlineExceeded, ""))),
				New(errTest, WithHTTPStatus(http.StatusBadGateway)),
			},
			wantHTTP: http.StatusInternalServerError,
			wantGRPC: codes.Unknown,
		},
		{
			name: "Neither 4xx nor 5xx",
			errors: []error{
				New(errTest, WithHTTPStatus(http.StatusPermanentRedirect)),
				New(errTest, WithHTTPStatus(http.StatusTemporaryRedirect)),
			},
			wantHTTP: http.StatusInternalServerError,
			wantGRPC: codes.Unknown,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			e := Combine(tt.errors...)
			ce := &Error{}

			if errors.As(e, &ce) {
				if tt.wantHTTP != 0 {
					status, ok := ce.HTTPStatus()
					require.True(t, ok)
					require.Equal(t, tt.wantHTTP, status)
				}

				if tt.wantGRPC != codes.OK {
					status, ok := ce.GRPCStatus()
					require.True(t, ok)
					require.Equal(t, tt.wantGRPC, status.Code())
				}
			} else {
				require.Fail(t, "Combine did not return an Error type")
			}
		})
	}
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

func TestError_HTTPStatus(t *testing.T) {
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
			name:    "No statuses set",
			hasCode: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := Error{
				error:      errTest,
				httpStatus: tt.httpStatus,
				grpcStatus: tt.grpcStatus,
			}

			s, ok := err.HTTPStatus()

			require.Equal(t, tt.hasCode, ok)
			require.Equal(t, tt.want, s)
		})
	}
}

func TestError_GRPCStatus(t *testing.T) {
	httpStatus := http.StatusTooManyRequests
	otherHTTPStatus := http.StatusOK
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
			httpStatus: otherHTTPStatus,
			grpcStatus: serverErr,
			want:       serverErr,
			hasCode:    true,
		},
		{
			name:    "No statuses set",
			hasCode: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := Error{
				error:      errTest,
				httpStatus: tt.httpStatus,
				grpcStatus: tt.grpcStatus,
			}

			s, ok := err.GRPCStatus()

			require.Equal(t, tt.hasCode, ok)
			require.Equal(t, tt.want, s)
		})
	}
}
