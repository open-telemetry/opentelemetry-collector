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
	"github.com/stretchr/testify/require"
)

func TestToServerOptions(t *testing.T) {
	// prepare
	oidcServer, err := newOIDCServer()
	require.NoError(t, err)
	oidcServer.Start()
	defer oidcServer.Close()

	config := Authentication{
		OIDC: &OIDC{
			IssuerURL:   oidcServer.URL,
			Audience:    "unit-test",
			GroupsClaim: "memberships",
		},
	}

	// test
	opts, err := config.ToServerOptions()

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, opts)
	assert.Len(t, opts, 2) // we have two interceptors
}

func TestInvalidConfigurationFailsOnToServerOptions(t *testing.T) {

	for _, tt := range []struct {
		cfg Authentication
	}{
		{
			Authentication{},
		},
		{
			Authentication{
				OIDC: &OIDC{
					IssuerURL:   "http://oidc.acme.invalid",
					Audience:    "unit-test",
					GroupsClaim: "memberships",
				},
			},
		},
	} {
		// test
		opts, err := tt.cfg.ToServerOptions()

		// verify
		assert.Error(t, err)
		assert.Nil(t, opts)
	}

}
