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

type defaultServer struct {
	ServerAuthenticateFunc
	component.StartFunc
	component.ShutdownFunc
}

// ServerOption represents the possible options for NewServer.
type ServerOption func(*defaultServer)

// ServerAuthenticateFunc defines the signature for the function responsible for performing the authentication based
// on the given headers map. See Server.Authenticate.
type ServerAuthenticateFunc func(ctx context.Context, headers map[string][]string) (context.Context, error)

func (f ServerAuthenticateFunc) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	if f == nil {
		return ctx, nil
	}
	return f(ctx, headers)
}

// WithServerAuthenticate specifies which function to use to perform the authentication.
func WithServerAuthenticate(authFunc ServerAuthenticateFunc) ServerOption {
	return func(o *defaultServer) {
		o.ServerAuthenticateFunc = authFunc
	}
}

// WithServerStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
func WithServerStart(startFunc component.StartFunc) ServerOption {
	return func(o *defaultServer) {
		o.StartFunc = startFunc
	}
}

// WithServerShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
func WithServerShutdown(shutdownFunc component.ShutdownFunc) ServerOption {
	return func(o *defaultServer) {
		o.ShutdownFunc = shutdownFunc
	}
}

// NewServer returns a Server configured with the provided options.
func NewServer(options ...ServerOption) Server {
	bc := &defaultServer{}

	for _, op := range options {
		op(bc)
	}

	return bc
}
