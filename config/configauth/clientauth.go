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

package configauth

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// ClientAuthenticator is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration.
type ClientAuthenticator interface {
	component.Extension
}

// HTTPClientAuthenticator is a ClientAuthenticator that can be used as an authenticator
// for the configauth.Authentication option for HTTP clients.
type HTTPClientAuthenticator interface {
	ClientAuthenticator
	RoundTripper(base http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClientAuthenticator is a ClientAuthenticator that can be used as an authenticator for
// the configauth.Authentication option for gRPC clients.
type GRPCClientAuthenticator interface {
	ClientAuthenticator
	PerRPCCredentials() (credentials.PerRPCCredentials, error)
}

// GetHTTPClientAuthenticator attempts to select the appropriate HTTPClientAuthenticator from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should be only used by HTTP clients.
func GetHTTPClientAuthenticator(extensions map[config.ComponentID]component.Extension,
	componentID config.ComponentID) (HTTPClientAuthenticator, error) {
	for id, ext := range extensions {
		if id == componentID {
			if auth, ok := ext.(HTTPClientAuthenticator); ok {
				return auth, nil
			}
			return nil, fmt.Errorf("requested authenticator is not for HTTP clients")
		}
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", componentID.String(), errAuthenticatorNotFound)
}

// GetGRPCClientAuthenticator attempts to select the appropriate GRPCClientAuthenticator from the list of extensions,
// based on the component id of the extension. If an authenticator is not found, an error is returned.
// This should only be used by gRPC clients.
func GetGRPCClientAuthenticator(extensions map[config.ComponentID]component.Extension,
	componentID config.ComponentID) (GRPCClientAuthenticator, error) {
	for id, ext := range extensions {
		if id == componentID {
			if auth, ok := ext.(GRPCClientAuthenticator); ok {
				return auth, nil
			}
			return nil, fmt.Errorf("requested authenticator is not for gRPC clients")
		}
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", componentID.String(), errAuthenticatorNotFound)
}
