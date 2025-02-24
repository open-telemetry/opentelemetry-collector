// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package auth // import "go.opentelemetry.io/collector/extension/auth"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Server is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration. Each Server is free to define its own behavior and configuration options,
// but note that the expectations that come as part of Extensions exist here as well. For instance, multiple instances of the same
// authenticator should be possible to exist under different names.
type Server interface {
	extension.Extension

	// Authenticate checks whether the given map contains valid auth data. Successfully authenticated calls will always return a nil error.
	// When the authentication fails, an error must be returned and the caller must not retry. This function is typically called from interceptors,
	// on behalf of receivers, but receivers can still call this directly if the usage of interceptors isn't suitable.
	// The deadline and cancellation given to this function must be respected, but note that authentication data has to be part of the map, not context.
	// The resulting context should contain the authentication data, such as the principal/username, group membership (if available), and the raw
	// authentication data (if possible). This will allow other components in the pipeline to make decisions based on that data, such as routing based
	// on tenancy as determined by the group membership, or passing through the authentication data to the next collector/backend.
	// The context keys to be used are not defined yet.
	Authenticate(ctx context.Context, sources map[string][]string) (context.Context, error)
}

type serverOptions struct {
	componentOptions []component.Option
	serverAuth       ServerAuthenticateFunc
}

// ServerOption represents the possible options for NewServer.
type ServerOption interface {
	apply(*serverOptions)
}

type serverOptionFunc func(*serverOptions)

func (of serverOptionFunc) apply(e *serverOptions) {
	of(e)
}

// ServerAuthenticateFunc defines the signature for the function responsible for performing the authentication based
// on the given sources map. See Server.Authenticate.
type ServerAuthenticateFunc func(ctx context.Context, sources map[string][]string) (context.Context, error)

func (f ServerAuthenticateFunc) Authenticate(ctx context.Context, sources map[string][]string) (context.Context, error) {
	if f == nil {
		return ctx, nil
	}
	return f(ctx, sources)
}

// WithServerAuthenticate specifies which function to use to perform the authentication.
func WithServerAuthenticate(authFunc ServerAuthenticateFunc) ServerOption {
	return serverOptionFunc(func(o *serverOptions) {
		o.serverAuth = authFunc
	})
}

// WithServerComponentOptions overrides the default component.Component (start/shutdown) functions for a Server.
// The default functions do nothing and always returns nil.
func WithServerComponentOptions(option ...component.Option) ServerOption {
	return serverOptionFunc(func(e *serverOptions) {
		e.componentOptions = append(e.componentOptions, option...)
	})
}

// Deprecated: [v0.121.0] use WithServerComponentOptions.
func WithServerStart(start component.StartFunc) ServerOption {
	return WithServerComponentOptions(component.WithStartFunc(start))
}

// Deprecated: [v0.121.0] use WithServerComponentOptions.
func WithServerShutdown(shutdown component.ShutdownFunc) ServerOption {
	return WithServerComponentOptions(component.WithShutdownFunc(shutdown))
}

type baseServer struct {
	component.Component
	ServerAuthenticateFunc
}

// NewServer returns a Server configured with the provided options.
func NewServer(options ...ServerOption) Server {
	so := &serverOptions{}

	for _, op := range options {
		op.apply(so)
	}

	return baseServer{
		Component:              component.NewComponent(so.componentOptions...),
		ServerAuthenticateFunc: so.serverAuth,
	}
}
