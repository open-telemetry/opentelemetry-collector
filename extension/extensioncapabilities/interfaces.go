// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package extensioncapabilities provides interfaces that can be implemented by extensions
// to provide additional capabilities.
package extensioncapabilities // import "go.opentelemetry.io/collector/extension/extensioncapabilities"

import (
	"context"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
)

// Dependent is an optional interface that can be implemented by extensions
// that depend on other extensions and must be started only after their dependencies.
// See https://github.com/open-telemetry/opentelemetry-collector/pull/8768 for examples.
type Dependent interface {
	extension.Extension
	Dependencies() []component.ID
}

// PipelineWatcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to pipeline
// states. Typically this will be used by extensions that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher interface {
	// Ready notifies the Extension that all pipelines were built and the
	// receivers were started, i.e.: the service is ready to receive data
	// (note that it may already have received data when this method is called).
	Ready() error

	// NotReady notifies the Extension that all receivers are about to be stopped,
	// i.e.: pipeline receivers will not accept new data.
	// This is sent before receivers are stopped, so the Extension can take any
	// appropriate actions before that happens.
	NotReady() error
}

// ConfigWatcher is an interface that should be implemented by an extension that
// wishes to be notified of the Collector's effective configuration.
type ConfigWatcher interface {
	// NotifyConfig notifies the extension of the Collector's current effective configuration.
	// The extension owns the `confmap.Conf`. Callers must ensure that it's safe for
	// extensions to store the `conf` pointer and use it concurrently with any other
	// instances of `conf`.
	NotifyConfig(ctx context.Context, conf *confmap.Conf) error
}

// HTTPClientAuthRoundTripper is an optional Extension interface that can be used as an HTTP authenticator for the configauth.Config option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the [configauth.Config] configuration.
type HTTPClientAuthRoundTripper interface {
	// RoundTripper returns a RoundTripper that can be used to authenticate HTTP requests.
	RoundTripper(base http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClientAuthenticator is an optional Extension interface that can be used as a gRPC authenticator for the configauth.Config option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the [configauth.Config] configuration.
type GRPCClientAuthenticator interface {
	// PerRPCCredentials returns a PerRPCCredentials that can be used to authenticate gRPC requests.
	PerRPCCredentials() (credentials.PerRPCCredentials, error)
}

// Authenticator is an optional Extension interface that can be used as an authenticator for the configauth.Config option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the [configauth.Config] configuration. Each Authenticator is free to define its own behavior and configuration options,
// but note that the expectations that come as part of Extensions exist here as well. For instance, multiple instances of the same
// authenticator should be possible to exist under different names.
type Authenticator interface {
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
