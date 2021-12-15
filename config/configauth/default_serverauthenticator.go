// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth // import "go.opentelemetry.io/collector/config/configauth"

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenthelper"
)

var _ ServerAuthenticator = (*defaultServerAuthenticator)(nil)

// Option represents the possible options for New.
type Option func(*defaultServerAuthenticator)

type defaultServerAuthenticator struct {
	AuthenticateFunc
	GRPCStreamInterceptorFunc
	GRPCUnaryInterceptorFunc
	HTTPInterceptorFunc

	componenthelper.StartFunc
	componenthelper.ShutdownFunc
}

// WithAuthenticate
func WithAuthenticate(authenticateFunc AuthenticateFunc) Option {
	return func(o *defaultServerAuthenticator) {
		o.AuthenticateFunc = authenticateFunc
	}
}

// WithGRPCStreamInterceptor
func WithGRPCStreamInterceptor(grpcStreamInterceptorFunc GRPCStreamInterceptorFunc) Option {
	return func(o *defaultServerAuthenticator) {
		o.GRPCStreamInterceptorFunc = grpcStreamInterceptorFunc
	}
}

// WithGRPCUnaryInterceptor
func WithGRPCUnaryInterceptor(grpcUnaryInterceptorFunc GRPCUnaryInterceptorFunc) Option {
	return func(o *defaultServerAuthenticator) {
		o.GRPCUnaryInterceptorFunc = grpcUnaryInterceptorFunc
	}
}

// WithHTTPInterceptor
func WithHTTPInterceptor(httpInterceptorFunc HTTPInterceptorFunc) Option {
	return func(o *defaultServerAuthenticator) {
		o.HTTPInterceptorFunc = httpInterceptorFunc
	}
}

// WithStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
func WithStart(startFunc componenthelper.StartFunc) Option {
	return func(o *defaultServerAuthenticator) {
		o.StartFunc = startFunc
	}
}

// WithShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
func WithShutdown(shutdownFunc componenthelper.ShutdownFunc) Option {
	return func(o *defaultServerAuthenticator) {
		o.ShutdownFunc = shutdownFunc
	}
}

// NewServerAuthenticator returns a ServerAuthenticator configured with the provided options.
func NewServerAuthenticator(options ...Option) ServerAuthenticator {
	bc := &defaultServerAuthenticator{
		AuthenticateFunc:          func(ctx context.Context, headers map[string][]string) (context.Context, error) { return ctx, nil },
		GRPCStreamInterceptorFunc: DefaultGRPCStreamServerInterceptor,
		GRPCUnaryInterceptorFunc:  DefaultGRPCUnaryServerInterceptor,
		HTTPInterceptorFunc:       DefaultHTTPInterceptor,

		StartFunc:    func(ctx context.Context, host component.Host) error { return nil },
		ShutdownFunc: func(ctx context.Context) error { return nil },
	}

	for _, op := range options {
		op(bc)
	}

	return bc
}

func (a *defaultServerAuthenticator) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	return a.AuthenticateFunc(ctx, headers)
}

func (a *defaultServerAuthenticator) GRPCStreamServerInterceptor(srv interface{}, stream grpc.ServerStream, srvInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return a.GRPCStreamInterceptorFunc(srv, stream, srvInfo, handler, a.AuthenticateFunc)
}

func (a *defaultServerAuthenticator) GRPCUnaryServerInterceptor(ctx context.Context, req interface{}, srvInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return a.GRPCUnaryInterceptorFunc(ctx, req, srvInfo, handler, a.AuthenticateFunc)
}

func (a *defaultServerAuthenticator) HTTPInterceptor(next http.Handler) http.Handler {
	return a.HTTPInterceptorFunc(next, a.AuthenticateFunc)
}

func (a *defaultServerAuthenticator) Start(ctx context.Context, host component.Host) error {
	return a.StartFunc(ctx, host)
}

func (a *defaultServerAuthenticator) Shutdown(ctx context.Context) error {
	return a.ShutdownFunc(ctx)
}
