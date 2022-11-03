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

package auth // import "go.opentelemetry.io/collector/extension/auth"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

var _ Server = (*defaultServer)(nil)

// Option represents the possible options for NewServer.
type Option func(*defaultServer)

type defaultServer struct {
	AuthenticateFunc
	component.StartFunc
	component.ShutdownFunc
}

// WithAuthenticate specifies which function to use to perform the authentication.
func WithAuthenticate(authenticateFunc AuthenticateFunc) Option {
	return func(o *defaultServer) {
		o.AuthenticateFunc = authenticateFunc
	}
}

// WithStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
func WithStart(startFunc component.StartFunc) Option {
	return func(o *defaultServer) {
		o.StartFunc = startFunc
	}
}

// WithShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
func WithShutdown(shutdownFunc component.ShutdownFunc) Option {
	return func(o *defaultServer) {
		o.ShutdownFunc = shutdownFunc
	}
}

// NewServer returns a Server configured with the provided options.
func NewServer(options ...Option) Server {
	bc := &defaultServer{
		AuthenticateFunc: func(ctx context.Context, headers map[string][]string) (context.Context, error) { return ctx, nil },
		StartFunc:        func(ctx context.Context, host component.Host) error { return nil },
		ShutdownFunc:     func(ctx context.Context) error { return nil },
	}

	for _, op := range options {
		op(bc)
	}

	return bc
}

// Authenticate performs the authentication.
func (a *defaultServer) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	return a.AuthenticateFunc(ctx, headers)
}

// Start the component.
func (a *defaultServer) Start(ctx context.Context, host component.Host) error {
	return a.StartFunc(ctx, host)
}

// Shutdown stops the component.
func (a *defaultServer) Shutdown(ctx context.Context) error {
	return a.ShutdownFunc(ctx)
}
