// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNetworkError_Error(t *testing.T) {
	tests := []struct {
		networkError error
		msgContains string
		validate    func(t *testing.T, se NetworkError)
	}{
		{
			networkError: NewHTTPError(errors.New("httperror"), 400),
			validate: func(t *testing.T, se NetworkError) {
				require.Contains(t, se.Error(), "400")
			},
		},
		{
			networkError: NewGRPCError(errors.New("status"), status.New(codes.InvalidArgument, "")),
			validate: func(t *testing.T, se NetworkError) {
				require.Contains(t, se.Error(), "InvalidArgument")
			},
		},
		{
			networkError: NewGRPCError(errors.New("nil"), nil),
			validate: func(t *testing.T, se NetworkError) {
				require.NotNil(t, se.Error())
			},
		},
	}
	for _, tt := range tests {
		var se NetworkError
		require.True(t, errors.As(tt.statusError, &se))
		tt.validate(t, se)
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	var err error = testErrorType{"some error"}
	se := NewHTTPError(err, 400)
	joinedErr := errors.Join(errors.New("other error"), se)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	require.True(t, errors.As(joinedErr, &target))
	require.Equal(t, err, target)
}

func TestNetworkError_GRPCStatus(t *testing.T) {
	tests := []struct {
		networkError error
		code        codes.Code
	}{
		{
			networkError: NewHTTPError(errors.New("httperror"), 429),
			code:        codes.ResourceExhausted,
		},
		{
			networkError: NewGRPCError(errors.New("status"), status.New(codes.ResourceExhausted, "")),
			code:        codes.ResourceExhausted,
		},
		{
			networkError: NewGRPCError(errors.New("nil"), nil),
			code:        codes.Unknown,
		},
	}
	for _, tt := range tests {
		var se NetworkError
		require.True(t, errors.As(tt.networkError, &se))
		status := se.GRPCStatus()
		require.Equal(t, tt.code, status.Code())
	}
}

func TestNetworkError_HTTPStatus(t *testing.T) {
	tests := []struct {
		networkError error
		code        int
		ok          bool
	}{
		{
			statusError: NewHTTPError(errors.New("httperror"), http.StatusTooManyRequests),
			code:        http.StatusTooManyRequests,
			ok:          true,
		},
		{
			networkError: NewGRPCError(errors.New("status"), status.New(codes.ResourceExhausted, "")),
			code:        http.StatusTooManyRequests,
			ok:          true,
		},
		{
			networkError: NewGRPCError(errors.New("nil"), nil),
			code:        http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		var se NetworkError
		require.True(t, errors.As(tt.networkError, &se))
		code := se.HTTPStatus()
		require.Equal(t, tt.code, code)
	}
}
