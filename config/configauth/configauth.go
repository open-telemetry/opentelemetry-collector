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
	errAuthenticatorNotFound  = errors.New("authenticator not found")
	errNotClientAuthenticator = errors.New("requested authenticator is not a client authenticator")
	errNotServerAuthenticator = errors.New("requested authenticator is not a server authenticator")
)

type PropagatePolicy struct {
	// Headers specifies a list of headers to propagate.
	// If empty, all headers are propagated.
	Headers []string `mapstructure:"headers"`

	// Query specifies a list of query parameters to propagate.
	// If empty, all query parameters are propagated.
	Query []string `mapstructure:"query"`
}

// Authentication defines the auth settings for the receiver.
type Authentication struct {
	// AuthenticatorID specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorID config.ComponentID `mapstructure:"authenticator"`

	// Propagate specifies the headers and query parameters to propagate to the corresponding auth extension. Optional.
	// If not specified, all headers and query params are propagated.
	Propagate PropagatePolicy `mapstructure:"propagate,omitempty"`
}

// GetServerAuthenticator attempts to select the appropriate ServerAuthenticator from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned.
func (a Authentication) GetServerAuthenticator(extensions map[config.ComponentID]component.Extension) (ServerAuthenticator, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if auth, ok := ext.(ServerAuthenticator); ok {
			return auth, nil
		}
		return nil, errNotServerAuthenticator
	}

	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}

// GetClientAuthenticator attempts to select the appropriate ClientAuthenticator from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should be only used by HTTP clients.
func (a Authentication) GetClientAuthenticator(extensions map[config.ComponentID]component.Extension) (ClientAuthenticator, error) {
	if ext, found := extensions[a.AuthenticatorID]; found {
		if auth, ok := ext.(ClientAuthenticator); ok {
			return auth, nil
		}
		return nil, errNotClientAuthenticator
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorID, errAuthenticatorNotFound)
}
