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

func TestGetServerAuthenticator(t *testing.T) {
	testCases := []struct {
		desc          string
		authenticator component.Extension
		expected      error
	}{
		{
			desc:          "obtain server authenticator",
			authenticator: NewServerAuthenticator(),
			expected:      nil,
		},
		{
			desc:          "not a server authenticator",
			authenticator: &MockClientAuthenticator{},
			expected:      errNotServerAuthenticator,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			cfg := &Authentication{
				AuthenticatorID: config.NewComponentID("mock"),
			}
			ext := map[config.ComponentID]component.Extension{
				config.NewComponentID("mock"): tC.authenticator,
			}

			authenticator, err := cfg.GetServerAuthenticator(ext)

			// verify
			if tC.expected != nil {
				assert.ErrorIs(t, err, tC.expected)
				assert.Nil(t, authenticator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authenticator)
			}
		})
	}
}

func TestGetServerAuthenticatorFails(t *testing.T) {
	cfg := &Authentication{
		AuthenticatorID: config.NewComponentID("does-not-exist"),
	}

	authenticator, err := cfg.GetServerAuthenticator(map[config.ComponentID]component.Extension{})
	assert.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}

func TestGetClientAuthenticator(t *testing.T) {
	testCases := []struct {
		desc          string
		authenticator component.Extension
		expected      error
	}{
		{
			desc:          "obtain client authenticator",
			authenticator: &MockClientAuthenticator{},
			expected:      nil,
		},
		{
			desc:          "not a client authenticator",
			authenticator: NewServerAuthenticator(),
			expected:      errNotClientAuthenticator,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			cfg := &Authentication{
				AuthenticatorID: config.NewComponentID("mock"),
			}
			ext := map[config.ComponentID]component.Extension{
				config.NewComponentID("mock"): tC.authenticator,
			}

			authenticator, err := cfg.GetClientAuthenticator(ext)

			// verify
			if tC.expected != nil {
				assert.ErrorIs(t, err, tC.expected)
				assert.Nil(t, authenticator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authenticator)
			}
		})
	}
}

func TestGetClientAuthenticatorFails(t *testing.T) {
	cfg := &Authentication{
		AuthenticatorID: config.NewComponentID("does-not-exist"),
	}
	authenticator, err := cfg.GetClientAuthenticator(map[config.ComponentID]component.Extension{})
	assert.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}
