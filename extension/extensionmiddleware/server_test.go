// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGetHTTPHandlerFunc(t *testing.T) {
	t.Run("nil_function", func(t *testing.T) {
		var f GetHTTPHandlerFunc
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		handler, err := f.GetHTTPHandler(baseHandler)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
		require.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("returns_wrapped_handler", func(t *testing.T) {
		called := false
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		f := GetHTTPHandlerFunc(func(base http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				base.ServeHTTP(w, r)
			}), nil
		})

		handler, err := f.GetHTTPHandler(baseHandler)
		require.NoError(t, err)
		require.NotNil(t, handler)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
		require.True(t, called)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("returns_error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		f := GetHTTPHandlerFunc(func(http.Handler) (http.Handler, error) {
			return nil, expectedErr
		})

		baseHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		handler, err := f.GetHTTPHandler(baseHandler)
		require.Equal(t, expectedErr, err)
		require.Nil(t, handler)
	})
}

func TestGetGRPCServerOptionsFunc(t *testing.T) {
	t.Run("nil_function", func(t *testing.T) {
		var f GetGRPCServerOptionsFunc
		opts, err := f.GetGRPCServerOptions()
		require.NoError(t, err)
		require.Nil(t, opts)
	})

	t.Run("returns_server_options", func(t *testing.T) {
		var interceptor grpc.UnaryServerInterceptor = func(
			context.Context,
			any,
			*grpc.UnaryServerInfo,
			grpc.UnaryHandler,
		) (resp any, err error) {
			return nil, nil
		}
		expectedOpts := []grpc.ServerOption{grpc.UnaryInterceptor(interceptor)}

		f := GetGRPCServerOptionsFunc(func() ([]grpc.ServerOption, error) {
			return expectedOpts, nil
		})

		opts, err := f.GetGRPCServerOptions()
		require.NoError(t, err)
		require.Equal(t, expectedOpts, opts)
	})

	t.Run("returns_error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		f := GetGRPCServerOptionsFunc(func() ([]grpc.ServerOption, error) {
			return nil, expectedErr
		})

		opts, err := f.GetGRPCServerOptions()
		require.Equal(t, expectedErr, err)
		require.Nil(t, opts)
	})
}
