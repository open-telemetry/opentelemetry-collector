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

package configoauth2

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestOAuth2ClientCredentialsSettings(t *testing.T) {
	tests := []struct {
		name        string
		settings    *OAuth2ClientCredentials
		shouldError bool
	}{
		{
			name: "all_valid_settings",
			settings: &OAuth2ClientCredentials{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError: false,
		},
		{
			name: "missing_client_id",
			settings: &OAuth2ClientCredentials{
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError: true,
		},
		{
			name: "missing_client_secret",
			settings: &OAuth2ClientCredentials{
				ClientID: "testclientid",
				TokenURL: "https://example.com/v1/token",
				Scopes:   []string{"resource.read"},
			},
			shouldError: true,
		},
		{
			name: "missing_token_url",
			settings: &OAuth2ClientCredentials{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				Scopes:       []string{"resource.read"},
			},
			shouldError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseRoundTripper := &testRoundTripperBase{""}
			_, err := test.settings.RoundTripper(baseRoundTripper)
			if test.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

type testRoundTripperBase struct {
	testString string
}

func (b *testRoundTripperBase) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestOAuth2ClientRoundTripperReturnsOAuth2RoundTripper(t *testing.T) {
	creds := &OAuth2ClientCredentials{
		ClientID:     "testclientid",
		ClientSecret: "testsecret",
		TokenURL:     "https://example.com/v1/token",
		Scopes:       []string{"resource.read"},
	}

	testString := "TestString"

	baseRoundTripper := &testRoundTripperBase{testString}
	roundTripper, err := creds.RoundTripper(baseRoundTripper)
	assert.NoError(t, err)

	// test roundTripper is an OAuth RoundTripper
	oAuth2Transport, ok := roundTripper.(*oauth2.Transport)
	assert.True(t, ok)

	// test oAuthRoundTripper wrapped the base roundTripper properly
	wrappedRoundTripper, ok := oAuth2Transport.Base.(*testRoundTripperBase)
	assert.True(t, ok)
	assert.Equal(t, wrappedRoundTripper.testString, testString)
}
