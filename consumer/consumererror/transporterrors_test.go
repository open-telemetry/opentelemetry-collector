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

func TestTransportError_Error(t *testing.T) {
	tests := []struct {
		transportError error
		msgContains    string
		validate       func(t *testing.T, se TransportError)
	}{
		{
			transportError: NewHTTP(errors.New("httperror"), 400),
			validate: func(t *testing.T, se TransportError) {
				require.Contains(t, se.Error(), "400")
			},
		},
		{
			transportError: NewGRPC(errors.New("status"), status.New(codes.InvalidArgument, "")),
			validate: func(t *testing.T, se TransportError) {
				require.Contains(t, se.Error(), "InvalidArgument")
			},
		},
		{
			transportError: NewGRPC(errors.New("nil"), nil),
			validate: func(t *testing.T, se TransportError) {
				require.NotNil(t, se.Error())
			},
		},
	}
	for _, tt := range tests {
		var se TransportError
		require.True(t, errors.As(tt.transportError, &se))
		tt.validate(t, se)
	}
}

func TestTransportError_Unwrap(t *testing.T) {
	var err error = testErrorType{"some error"}
	se := NewHTTP(err, 400)
	joinedErr := errors.Join(errors.New("other error"), se)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	require.True(t, errors.As(joinedErr, &target))
	require.Equal(t, err, target)
}

func TestTransportError_GRPCStatus(t *testing.T) {
	tests := []struct {
		transportError error
		code           codes.Code
	}{
		{
			transportError: NewHTTP(errors.New("httperror"), 429),
			code:           codes.ResourceExhausted,
		},
		{
			transportError: NewGRPC(errors.New("status"), status.New(codes.ResourceExhausted, "")),
			code:           codes.ResourceExhausted,
		},
		{
			transportError: NewGRPC(errors.New("nil"), nil),
			code:           codes.Unknown,
		},
	}
	for _, tt := range tests {
		var se TransportError
		require.True(t, errors.As(tt.transportError, &se))
		status := se.GRPCStatus()
		require.Equal(t, tt.code, status.Code())
	}
}

func TestTransportError_HTTPStatus(t *testing.T) {
	tests := []struct {
		transportError error
		code           int
		ok             bool
	}{
		{
			transportError: NewHTTP(errors.New("httperror"), http.StatusTooManyRequests),
			code:           http.StatusTooManyRequests,
			ok:             true,
		},
		{
			transportError: NewGRPC(errors.New("status"), status.New(codes.ResourceExhausted, "")),
			code:           http.StatusTooManyRequests,
			ok:             true,
		},
		{
			transportError: NewGRPC(errors.New("nil"), nil),
			code:           http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		var se TransportError
		require.True(t, errors.As(tt.transportError, &se))
		code := se.HTTPStatus()
		require.Equal(t, tt.code, code)
	}
}
