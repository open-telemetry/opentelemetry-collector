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

// Package configauth implements the configuration settings to
// ensure authentication on incoming requests, and allows
// exporters to add authentication on outgoing requests.
package configauth // import "go.opentelemetry.io/collector/config/configauth"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/auth"
)

var (
	errAuthenticatorNotFound = errors.New("authenticator not found")
	errNotClient             = errors.New("requested authenticator is not a client authenticator")
	errNotServer             = errors.New("requested authenticator is not a server authenticator")
)

// Authentication defines the auth settings for the receiver.
type Authentication struct {
	// AuthenticatorID specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorID component.ID `mapstructure:"authenticator"`
}

// GetServerAuthenticator attempts to select the appropriate auth.Server from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned.
func (a Authentication) GetServerAuthenticator(extensions map[component.ID]component.Component) (auth.Server, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if server, ok := ext.(auth.Server); ok {
			return server, nil
		}
		return nil, errNotServer
	}

	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetClientAuthenticator attempts to select the appropriate auth.Client from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should be only used by HTTP clients.
func (a Authentication) GetClientAuthenticator(extensions map[component.ID]component.Component) (auth.Client, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if client, ok := ext.(auth.Client); ok {
			return client, nil
		}
		return nil, errNotClient
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}
