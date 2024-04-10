// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusError_Error(t *testing.T) {
	tests := []struct {
		statusError error
		msgContains string
		validate    func(t *testing.T, se *StatusError)
	}{
		{
			statusError: NewHTTPStatus(errors.New("httperror"), 400),
			validate: func(t *testing.T, se *StatusError) {
				require.Contains(t, se.Error(), "400")
			},
		},
		{
			statusError: NewGRPCStatus(errors.New("status"), status.New(codes.InvalidArgument, "")),
			validate: func(t *testing.T, se *StatusError) {
				require.Contains(t, se.Error(), "InvalidArgument")
			},
		},
		{
			statusError: &StatusError{},
			validate: func(t *testing.T, se *StatusError) {
				require.NotNil(t, se.Error())
			},
		},
	}
	for _, tt := range tests {
		se := &StatusError{}
		require.True(t, errors.As(tt.statusError, &se))
		tt.validate(t, se)
	}
}

func TestStatusError_Unwrap(t *testing.T) {
	var err error = testErrorType{"some error"}
	se := NewHTTPStatus(err, 400)
	joinedErr := errors.Join(errors.New("other error"), se)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	require.True(t, errors.As(joinedErr, &target))
	require.Equal(t, err, target)
}

func TestStatusError_GRPCStatus(t *testing.T) {
	tests := []struct {
		statusError error
		code        codes.Code
		ok          bool
	}{
		{
			statusError: NewHTTPStatus(errors.New("httperror"), 429),
			code:        codes.ResourceExhausted,
			ok:          true,
		},
		{
			statusError: NewGRPCStatus(errors.New("status"), status.New(codes.ResourceExhausted, "")),
			code:        codes.ResourceExhausted,
			ok:          true,
		},
		{
			statusError: &StatusError{},
			ok:          false,
		},
	}
	for _, tt := range tests {
		se := &StatusError{}
		require.True(t, errors.As(tt.statusError, &se))
		status, ok := se.GRPCStatus()
		require.Equal(t, tt.ok, ok)
		if ok {
			require.Equal(t, tt.code, status.Code())
		}
	}
}

func TestStatusError_HTTPStatus(t *testing.T) {
	tests := []struct {
		statusError error
		code        int
		ok          bool
	}{
		{
			statusError: NewHTTPStatus(errors.New("httperror"), 429),
			code:        429,
			ok:          true,
		},
		{
			statusError: NewGRPCStatus(errors.New("status"), status.New(codes.ResourceExhausted, "")),
			code:        429,
			ok:          true,
		},
		{
			statusError: &StatusError{},
			ok:          false,
		},
	}
	for _, tt := range tests {
		se := &StatusError{}
		require.True(t, errors.As(tt.statusError, &se))
		code, ok := se.HTTPStatus()
		require.Equal(t, tt.ok, ok)
		if ok {
			require.Equal(t, tt.code, code)
		}
	}
}
