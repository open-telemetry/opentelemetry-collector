// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGetHTTPRoundTripperFunc(t *testing.T) {
	// Create a base round tripper for testing
	baseRT := http.DefaultTransport

	t.Run("nil function", func(t *testing.T) {
		var nilFunc GetHTTPRoundTripperFunc
		rt, err := nilFunc.GetHTTPRoundTripper(baseRT)
		require.NoError(t, err)
		require.Equal(t, baseRT, rt)
	})

	t.Run("identity function", func(t *testing.T) {
		identityFunc := GetHTTPRoundTripperFunc(func(base http.RoundTripper) (http.RoundTripper, error) {
			return base, nil
		})
		rt, err := identityFunc.GetHTTPRoundTripper(baseRT)
		require.NoError(t, err)
		require.Equal(t, baseRT, rt)
	})

	t.Run("error function", func(t *testing.T) {
		expectedErr := errors.New("round tripper error")
		errorFunc := GetHTTPRoundTripperFunc(func(_ http.RoundTripper) (http.RoundTripper, error) {
			return nil, expectedErr
		})

		rt, err := errorFunc.GetHTTPRoundTripper(baseRT)
		require.Error(t, err)
		require.Equal(t, expectedErr, err)
		require.Nil(t, rt)
	})
}

func TestGetGRPCClientOptionsFunc(t *testing.T) {
	t.Run("nil function", func(t *testing.T) {
		var nilFunc GetGRPCClientOptionsFunc
		options, err := nilFunc.GetGRPCClientOptions()
		require.NoError(t, err)
		require.Nil(t, options)
	})

	t.Run("empty options function", func(t *testing.T) {
		emptyFunc := GetGRPCClientOptionsFunc(func() ([]grpc.DialOption, error) {
			return []grpc.DialOption{}, nil
		})

		options, err := emptyFunc.GetGRPCClientOptions()
		require.NoError(t, err)
		require.Empty(t, options)
	})

	t.Run("options function", func(t *testing.T) {
		// Create some test dial options
		dialOpt1 := grpc.WithAuthority("test-authority")
		dialOpt2 := grpc.WithDisableRetry()

		optionsFunc := GetGRPCClientOptionsFunc(func() ([]grpc.DialOption, error) {
			return []grpc.DialOption{dialOpt1, dialOpt2}, nil
		})

		options, err := optionsFunc.GetGRPCClientOptions()
		require.NoError(t, err)
		require.Len(t, options, 2)
	})

	t.Run("error function", func(t *testing.T) {
		expectedErr := errors.New("grpc options error")
		errorFunc := GetGRPCClientOptionsFunc(func() ([]grpc.DialOption, error) {
			return nil, expectedErr
		})

		options, err := errorFunc.GetGRPCClientOptions()
		require.Error(t, err)
		require.Equal(t, expectedErr, err)
		require.Nil(t, options)
	})
}
