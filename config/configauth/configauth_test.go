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
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/auth"
)

func TestGetServer(t *testing.T) {
	testCases := []struct {
		desc          string
		authenticator extension.Extension
		expected      error
	}{
		{
			desc:          "obtain server authenticator",
			authenticator: auth.NewServer(),
			expected:      nil,
		},
		{
			desc:          "not a server authenticator",
			authenticator: auth.NewClient(),
			expected:      errNotServer,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			cfg := &Authentication{
				AuthenticatorID: component.NewID("mock"),
			}
			ext := map[component.ID]component.Component{
				component.NewID("mock"): tC.authenticator,
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

func TestGetServerFails(t *testing.T) {
	cfg := &Authentication{
		AuthenticatorID: component.NewID("does-not-exist"),
	}

	authenticator, err := cfg.GetServerAuthenticator(map[component.ID]component.Component{})
	assert.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}

func TestGetClient(t *testing.T) {
	testCases := []struct {
		desc          string
		authenticator extension.Extension
		expected      error
	}{
		{
			desc:          "obtain client authenticator",
			authenticator: auth.NewClient(),
			expected:      nil,
		},
		{
			desc:          "not a client authenticator",
			authenticator: auth.NewServer(),
			expected:      errNotClient,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			cfg := &Authentication{
				AuthenticatorID: component.NewID("mock"),
			}
			ext := map[component.ID]component.Component{
				component.NewID("mock"): tC.authenticator,
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

func TestGetClientFails(t *testing.T) {
	cfg := &Authentication{
		AuthenticatorID: component.NewID("does-not-exist"),
	}
	authenticator, err := cfg.GetClientAuthenticator(map[component.ID]component.Component{})
	assert.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}
