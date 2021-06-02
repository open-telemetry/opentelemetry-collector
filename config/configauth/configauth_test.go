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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

func TestGetAuthenticator(t *testing.T) {
	// prepare
	cfg := &Authentication{
		AuthenticatorName: "mock",
	}
	ext := map[config.ComponentID]component.Extension{
		config.NewID("mock"): &MockAuthenticator{},
	}

	// test
	componentID, err := config.NewIDFromString(cfg.AuthenticatorName)
	assert.NoError(t, err)

	authenticator, err := GetServerAuthenticator(ext, componentID)

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, authenticator)
}

func TestGetAuthenticatorFails(t *testing.T) {
	testCases := []struct {
		desc     string
		cfg      *Authentication
		ext      map[config.ComponentID]component.Extension
		expected error
	}{
		{
			desc: "ServerAuthenticator not found",
			cfg: &Authentication{
				AuthenticatorName: "does-not-exist",
			},
			ext:      map[config.ComponentID]component.Extension{},
			expected: errAuthenticatorNotFound,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			componentID, err := config.NewIDFromString(tC.cfg.AuthenticatorName)
			assert.NoError(t, err)
			authenticator, err := GetServerAuthenticator(tC.ext, componentID)
			assert.ErrorIs(t, err, tC.expected)
			assert.Nil(t, authenticator)
		})
	}
}
