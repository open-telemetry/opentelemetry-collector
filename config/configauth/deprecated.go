// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth // import "go.opentelemetry.io/collector/config/configauth"

import (
	"go.opentelemetry.io/collector/extension/auth"
	"go.opentelemetry.io/collector/extension/auth/authtest"
)

// Deprecated: [v0.67.0] Use auth.Client
type ClientAuthenticator = auth.Client

// Option represents the possible options for NewServerAuthenticator.
// Deprecated: [v0.67.0] Use auth.ClientOption
type ClientOption = auth.ClientOption

// WithClientStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.67.0] Use auth.WithClientStart
var WithClientStart = auth.WithClientStart

// WithClientShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.67.0] Use auth.WithClientShutdown
var WithClientShutdown = auth.WithClientShutdown

// WithClientRoundTripper provides a `RoundTripper` function for this client authenticator.
// The default round tripper is no-op.
// Deprecated: [v0.67.0] Use auth.WithClientRoundTripper
var WithClientRoundTripper = auth.WithClientRoundTripper

// WithPerRPCCredentials provides a `PerRPCCredentials` function for this client authenticator.
// There's no default.
// Deprecated: [v0.67.0] Use auth.WithPerRPCCredentials
var WithPerRPCCredentials = auth.WithPerRPCCredentials

// NewClientAuthenticator returns a ClientAuthenticator configured with the provided options.
// Deprecated: [v0.67.0] Use auth.NewClient
var NewClientAuthenticator = auth.NewClient

// Deprecated: [v0.67.0] Use auth.Server
type ServerAuthenticator = auth.Server

// Deprecated: [v0.67.0] Use auth.AuthenticateFunc
type AuthenticateFunc = auth.AuthenticateFunc

// Option represents the possible options for NewServerAuthenticator.
// Deprecated: [v0.67.0] Use auth.Option
type Option = auth.Option

// WithAuthenticate specifies which function to use to perform the authentication.
// Deprecated: [v0.67.0] Use auth.WithAuthenticate
var WithAuthenticate = auth.WithAuthenticate

// WithStart overrides the default `Start` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.67.0] Use auth.WithStart
var WithStart = auth.WithStart

// WithShutdown overrides the default `Shutdown` function for a component.Component.
// The default always returns nil.
// Deprecated: [v0.67.0] Use auth.WithShutdown
var WithShutdown = auth.WithShutdown

// NewServerAuthenticator returns a ServerAuthenticator configured with the provided options.
// Deprecated: [v0.67.0] Use auth.NewServer
var NewServerAuthenticator = auth.NewServer

// Deprecated: [v0.67.0] Use auth.MockClient
type MockClientAuthenticator = authtest.MockClient
