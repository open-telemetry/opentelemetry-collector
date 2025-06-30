// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth // import "go.opentelemetry.io/collector/extension/extensionauth"

import (
	"context"
)

// Server is an optional Extension interface that can be used as an authenticator for the configauth.Config option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the [configauth.Config] configuration. Each Server is free to define its own behavior and configuration options,
// but note that the expectations that come as part of Extensions exist here as well. For instance, multiple instances of the same
// authenticator should be possible to exist under different names.
type Server interface {
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

// ServerAuthenticateFunc defines the signature for the function responsible for performing the authentication based
// on the given sources map. See Server.Authenticate.
type ServerAuthenticateFunc func(ctx context.Context, sources map[string][]string) (context.Context, error)

func (f ServerAuthenticateFunc) Authenticate(ctx context.Context, sources map[string][]string) (context.Context, error) {
	if f == nil {
		return ctx, nil
	}
	return f(ctx, sources)
}
