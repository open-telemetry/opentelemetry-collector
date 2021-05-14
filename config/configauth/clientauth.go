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

// ClientAuth is an Extension that can be used as an authenticator for the configauth.Authentication option.
// Authenticators are then included as part of OpenTelemetry Collector builds and can be referenced by their
// names from the Authentication configuration. .
type ClientAuth interface {
	component.Extension
}

// HTTPClientAuth is a ClientAuth that can be used as an authenticator for the configauth.Authentication option for HTTP
// clients.
type HTTPClientAuth interface {
	ClientAuth
	RoundTripper(base http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClientAuth is a ClientAuth that can be used as an authenticator for the configauth.Authentication option for gRPC
// clients.
type GRPCClientAuth interface {
	ClientAuth
	PerRPCCredential() (credentials.PerRPCCredentials, error)
}

// GetHTTPClientAuth attempts to select the appropriate HTTPClientAuth from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned. This should be only
// used by HTTP clients.
func GetHTTPClientAuth(extensions map[config.ComponentID]component.Extension, requested string) (HTTPClientAuth, error) {
	if requested == "" {
		return nil, errAuthenticatorNotProvided
	}

	reqID, err := config.NewIDFromString(requested)
	if err != nil {
		return nil, err
	}

	for name, ext := range extensions {
		if name == reqID {
			if auth, ok := ext.(HTTPClientAuth); ok {
				return auth, nil
			}
			return nil, fmt.Errorf("requested authenticator is not for HTTP clients")
		}
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", requested, errAuthenticatorNotFound)
}

// GetGRPCClientAuth attempts to select the appropriate GRPCClientAuth from the list of extensions,
// based on the requested extension name. If an authenticator is not found, an error is returned. This shold only be used
// by gRPC clients
func GetGRPCClientAuth(extensions map[config.ComponentID]component.Extension, requested string) (GRPCClientAuth, error) {
	if requested == "" {
		return nil, errAuthenticatorNotProvided
	}

	reqID, err := config.NewIDFromString(requested)
	if err != nil {
		return nil, err
	}

	for name, ext := range extensions {
		if name == reqID {
			if auth, ok := ext.(GRPCClientAuth); ok {
				return auth, nil
			}
			return nil, fmt.Errorf("requested authenticator is not for gRPC clients")
		}
	}
	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", requested, errAuthenticatorNotFound)
}
