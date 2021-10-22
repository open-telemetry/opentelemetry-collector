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
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

var (
	errAuthenticatorNotFound = errors.New("authenticator not found")
)

// Authentication defines the auth settings for the receiver.
type Authentication struct {
	// AuthenticatorID specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorID config.ComponentID `mapstructure:"authenticator"`
}

// GetServerAuthenticator attempts to select the appropriate ServerAuthenticator from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned.
func (a Authentication) GetServerAuthenticator(extensions map[config.ComponentID]component.Extension) (ServerAuthenticator, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if auth, ok := ext.(ServerAuthenticator); ok {
			return auth, nil
		}
		return nil, fmt.Errorf("requested authenticator is not a server authenticator")
	}

	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetHTTPClientAuthenticator attempts to select the appropriate HTTPClientAuthenticator from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should be only used by HTTP clients.
func (a Authentication) GetHTTPClientAuthenticator(extensions map[config.ComponentID]component.Extension) (HTTPClientAuthenticator, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if auth, ok := ext.(HTTPClientAuthenticator); ok {
			return auth, nil
		}
		return nil, fmt.Errorf("requested authenticator is not a HTTPClientAuthenticator")
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetGRPCClientAuthenticator attempts to select the appropriate GRPCClientAuthenticator from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should only be used by gRPC clients.
func (a Authentication) GetGRPCClientAuthenticator(extensions map[config.ComponentID]component.Extension) (GRPCClientAuthenticator, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if auth, ok := ext.(GRPCClientAuthenticator); ok {
			return auth, nil
		}
		return nil, fmt.Errorf("requested authenticator is not a GRPCClientAuthenticator")
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}
