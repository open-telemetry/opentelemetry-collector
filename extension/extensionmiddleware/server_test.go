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
	testctx := context.Background()

	t.Run("nil_function", func(t *testing.T) {
		var f GetHTTPHandlerFunc
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		hfunc, err := f.GetHTTPHandler(testctx)
		require.NoError(t, err)

		handler, err := hfunc(testctx, baseHandler)
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

		f := GetHTTPHandlerFunc(func(_ context.Context) (WrapHTTPHandlerFunc, error) {
			return func(_ context.Context, base http.Handler) (http.Handler, error) {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called = true
					base.ServeHTTP(w, r)
				}), nil
			}, nil
		})

		hfunc, err := f.GetHTTPHandler(testctx)
		require.NoError(t, err)
		require.NotNil(t, hfunc)

		handler, err := hfunc(testctx, baseHandler)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
		require.True(t, called)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("returns_error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		f := GetHTTPHandlerFunc(func(context.Context) (WrapHTTPHandlerFunc, error) {
			return nil, expectedErr
		})

		hfunc, err := f.GetHTTPHandler(testctx)
		require.Equal(t, expectedErr, err)
		require.Nil(t, hfunc)
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
