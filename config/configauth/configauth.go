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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

var (
	errAuthenticatorNotFound = errors.New("authenticator not found")
)

// Authentication defines the auth settings for the receiver.
type Authentication struct {
	// AuthenticatorName specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorName string `mapstructure:"authenticator"`
}

// GetServerAuthenticator attempts to select the appropriate from the list of extensions, based on the requested extension name.
// If an authenticator is not found, an error is returned.
func GetServerAuthenticator(extensions map[config.ComponentID]component.Extension, componentID config.ComponentID) (ServerAuthenticator, error) {
	for id, ext := range extensions {
		if auth, ok := ext.(ServerAuthenticator); ok {
			if id == componentID {
				return auth, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", componentID.String(), errAuthenticatorNotFound)
}
