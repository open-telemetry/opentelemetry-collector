// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth // import "go.opentelemetry.io/collector/extension/extensionauth"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Server is an optional Extension interface that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration. Each Server is free to define its own behavior and configuration options,
// but note that the expectations that come as part of Extensions exist here as well. For instance, multiple instances of the same
// authenticator should be possible to exist under different names.
type Server interface {
	// Deprecated: will be removed in the next release.
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

var _ Server = (*defaultServer)(nil)

type defaultServer struct {
	serverAuthenticateFunc ServerAuthenticateFunc
	component.StartFunc
	component.ShutdownFunc
}

// Authenticate implements Server.
func (d *defaultServer) Authenticate(ctx context.Context, sources map[string][]string) (context.Context, error) {
	if d.serverAuthenticateFunc == nil {
		return ctx, nil
	}
	return d.serverAuthenticateFunc(ctx, sources)
}

// ServerOption represents the possible options for NewServer.
type ServerOption interface {
	apply(*defaultServer)
}

type serverOptionFunc func(*defaultServer)

func (of serverOptionFunc) apply(e *defaultServer) {
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
// Deprecated: [v0.122.0] This type is deprecated and will be removed in the next release.
func WithServerAuthenticate(authFunc ServerAuthenticateFunc) ServerOption {
	return serverOptionFunc(func(o *defaultServer) {
		o.serverAuthenticateFunc = authFunc
	})
}

// WithServerStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.122.0] This type is deprecated and will be removed in the next release.
func WithServerStart(startFunc component.StartFunc) ServerOption {
	return serverOptionFunc(func(o *defaultServer) {
		o.StartFunc = startFunc
	})
}

// WithServerShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.122.0] This type is deprecated and will be removed in the next release.
func WithServerShutdown(shutdownFunc component.ShutdownFunc) ServerOption {
	return serverOptionFunc(func(o *defaultServer) {
		o.ShutdownFunc = shutdownFunc
	})
}

// NewServer returns a Server configured with the provided options.
// Deprecated: [v0.122.0] This type is deprecated and will be removed in the next release.
// Manually implement the [Server] interface instead.
func NewServer(options ...ServerOption) (Server, error) {
	bc := &defaultServer{}

	for _, op := range options {
		op.apply(bc)
	}

	return bc, nil
}
