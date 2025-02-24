// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package auth // import "go.opentelemetry.io/collector/extension/auth"

import (
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Client is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration.
type Client interface {
	extension.Extension

	// RoundTripper returns a RoundTripper that can be used to authenticate HTTP requests.
	RoundTripper(base http.RoundTripper) (http.RoundTripper, error)

	// PerRPCCredentials returns a PerRPCCredentials that can be used to authenticate gRPC requests.
	PerRPCCredentials() (credentials.PerRPCCredentials, error)
}

// ClientOption represents the possible options for NewClient.
type ClientOption interface {
	apply(*clientOptions)
}

type clientOptionFunc func(*clientOptions)

func (of clientOptionFunc) apply(e *clientOptions) {
	of(e)
}

// ClientRoundTripperFunc specifies the function that returns a RoundTripper that can be used to authenticate HTTP requests.
type ClientRoundTripperFunc func(base http.RoundTripper) (http.RoundTripper, error)

func (f ClientRoundTripperFunc) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if f == nil {
		return base, nil
	}
	return f(base)
}

// ClientPerRPCCredentialsFunc specifies the function that returns a PerRPCCredentials that can be used to authenticate gRPC requests.
type ClientPerRPCCredentialsFunc func() (credentials.PerRPCCredentials, error)

func (f ClientPerRPCCredentialsFunc) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	if f == nil {
		return nil, nil
	}
	return f()
}

type clientOptions struct {
	componentOptions  []component.Option
	roundTripper      ClientRoundTripperFunc
	perRPCCredentials ClientPerRPCCredentialsFunc
}

// WithClientComponentOptions overrides the default component.Component (start/shutdown) functions for a Client.
// The default functions do nothing and always returns nil.
func WithClientComponentOptions(option ...component.Option) ClientOption {
	return clientOptionFunc(func(e *clientOptions) {
		e.componentOptions = append(e.componentOptions, option...)
	})
}

// Deprecated: [v0.121.0] use WithClientComponentOptions.
func WithClientStart(start component.StartFunc) ClientOption {
	return WithClientComponentOptions(component.WithStartFunc(start))
}

// Deprecated: [v0.121.0] use WithClientComponentOptions.
func WithClientShutdown(shutdown component.ShutdownFunc) ClientOption {
	return WithClientComponentOptions(component.WithShutdownFunc(shutdown))
}

// WithClientRoundTripper provides a `RoundTripper` function for this client authenticator.
// The default round tripper is no-op.
func WithClientRoundTripper(roundTripperFunc ClientRoundTripperFunc) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.roundTripper = roundTripperFunc
	})
}

// WithClientPerRPCCredentials provides a `PerRPCCredentials` function for this client authenticator.
// There's no default.
func WithClientPerRPCCredentials(perRPCCredentialsFunc ClientPerRPCCredentialsFunc) ClientOption {
	return clientOptionFunc(func(o *clientOptions) {
		o.perRPCCredentials = perRPCCredentialsFunc
	})
}

type baseClient struct {
	component.Component
	ClientRoundTripperFunc
	ClientPerRPCCredentialsFunc
}

// NewClient returns a Client configured with the provided options.
func NewClient(options ...ClientOption) Client {
	bc := &clientOptions{}

	for _, op := range options {
		op.apply(bc)
	}

	return baseClient{
		Component:                   component.NewComponent(bc.componentOptions...),
		ClientRoundTripperFunc:      bc.roundTripper,
		ClientPerRPCCredentialsFunc: bc.perRPCCredentials,
	}
}
