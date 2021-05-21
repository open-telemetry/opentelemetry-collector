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
	errAuthenticatorNotFound    = errors.New("authenticator not found")
	errAuthenticatorNotProvided = errors.New("authenticator not provided")
)

// Authentication defines the auth settings for the receiver.
type Authentication struct {
	// Authenticator specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorName string `mapstructure:"authenticator"`
}

<<<<<<< HEAD
// GetAuthenticator attempts to select the appropriate from the list of extensions, based on the requested extension name.
// If an authenticator is not found, an error is returned.
func GetAuthenticator(extensions map[config.ComponentID]component.Extension, requested string) (Authenticator, error) {
	if requested == "" {
		return nil, errAuthenticatorNotProvided
	}

	reqID, err := config.NewIDFromString(requested)
	if err != nil {
		return nil, err
	}

=======
// GetServerAuthenticator attempts to select the appropriate ServerAuthenticator from the list of extensions, based on the requested extension name.
// If authenticator is not found, an error is returned.
func GetServerAuthenticator(extensions map[config.ComponentID]component.Extension, componentID config.ComponentID) (ServerAuthenticator, error) {
>>>>>>> bf891180 (Fix typos in Config Package)
	for name, ext := range extensions {
		if auth, ok := ext.(Authenticator); ok {
			if name == reqID {
				return auth, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to resolve authenticator %q: %w", requested, errAuthenticatorNotFound)
}
