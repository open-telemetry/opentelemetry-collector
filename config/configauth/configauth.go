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

package configauth

import (
	"errors"
	"fmt"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

var (
	errAuthenticatorNotFound    error = errors.New("authenticator not found")
	errAuthenticatorNotProvided error = errors.New("authenticator not provided")
)

// Authentication defines the auth settings for the receiver
type Authentication struct {
	// Authenticator specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorName string `mapstructure:"authenticator"`
}

// ToServerOption builds a set of server options ready to be used by the gRPC server
func (a *Authentication) ToServerOption(ext map[config.NamedEntity]component.Extension) ([]grpc.ServerOption, error) {
	if a.AuthenticatorName == "" {
		return nil, errAuthenticatorNotProvided
	}

	authenticator := selectAuthenticator(ext, a.AuthenticatorName)
	if authenticator == nil {
		return nil, fmt.Errorf("failed to resolve authenticator %q: %w", a.AuthenticatorName, errAuthenticatorNotFound)
	}

	return []grpc.ServerOption{
		grpc.UnaryInterceptor(authenticator.UnaryInterceptor),
		grpc.StreamInterceptor(authenticator.StreamInterceptor),
	}, nil
}

func selectAuthenticator(extensions map[config.NamedEntity]component.Extension, requested string) Authenticator {
	for name, ext := range extensions {
		if auth, ok := ext.(Authenticator); ok {
			if name.Name() == requested {
				return auth
			}
		}
	}
	return nil
}
