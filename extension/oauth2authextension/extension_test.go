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

package oauth2authextension

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	grpcOAuth "google.golang.org/grpc/credentials/oauth"
)

func TestOAuthClientSettings(t *testing.T) {
	tests := []struct {
		name          string
		settings      *OAuth2ClientSettings
		shouldError   bool
		expectedError error
	}{
		{
			name: "all_valid_settings",
			settings: &OAuth2ClientSettings{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError:   false,
			expectedError: nil,
		},
		{
			name: "missing_client_id",
			settings: &OAuth2ClientSettings{
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError:   true,
			expectedError: errNoClientIDProvided,
		},
		{
			name: "missing_client_secret",
			settings: &OAuth2ClientSettings{
				ClientID: "testclientid",
				TokenURL: "https://example.com/v1/token",
				Scopes:   []string{"resource.read"},
			},
			shouldError:   true,
			expectedError: errNoClientSecretProvided,
		},
		{
			name: "missing_token_url",
			settings: &OAuth2ClientSettings{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				Scopes:       []string{"resource.read"},
			},
			shouldError:   true,
			expectedError: errNoTokenURLProvided,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rc, err := newOAuth2Extension(test.settings, zap.NewNop())
			if test.shouldError {
				assert.Error(t, err)
				assert.Equal(t, test.expectedError, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.settings.Scopes, rc.clientCredentials.Scopes)
			assert.Equal(t, test.settings.TokenURL, rc.clientCredentials.TokenURL)
			assert.Equal(t, test.settings.ClientSecret, rc.clientCredentials.ClientSecret)
			assert.Equal(t, test.settings.ClientID, rc.clientCredentials.ClientID)
		})
	}
}

type testRoundTripper struct {
	testString string
}

func (b *testRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestRoundTripper(t *testing.T) {
	tests := []struct {
		name        string
		settings    *OAuth2ClientSettings
		shouldError bool
	}{
		{
			name: "returns_http_round_tripper",
			settings: &OAuth2ClientSettings{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError: false,
		},
		{
			name: "invalid_client_settings_should_error",
			settings: &OAuth2ClientSettings{
				ClientID: "testclientid",
				TokenURL: "https://example.com/v1/token",
				Scopes:   []string{"resource.read"},
			},
			shouldError: true,
		},
	}

	testString := "TestString"
	baseRoundTripper := &testRoundTripper{testString}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			oauth2Authenticator, err := newOAuth2Extension(testcase.settings, zap.NewNop())
			if testcase.shouldError {
				assert.Error(t, err)
				assert.Nil(t, oauth2Authenticator)
				return
			}

			assert.NotNil(t, oauth2Authenticator)
			roundTripper, err := oauth2Authenticator.RoundTripper(baseRoundTripper)
			assert.Nil(t, err)

			// test roundTripper is an OAuth RoundTripper
			oAuth2Transport, ok := roundTripper.(*oauth2.Transport)
			assert.True(t, ok)

			// test oAuthRoundTripper wrapped the base roundTripper properly
			wrappedRoundTripper, ok := oAuth2Transport.Base.(*testRoundTripper)
			assert.True(t, ok)
			assert.Equal(t, wrappedRoundTripper.testString, testString)
		})
	}
}

func TestOAuth2PerRPCCredentials(t *testing.T) {
	tests := []struct {
		name        string
		settings    *OAuth2ClientSettings
		shouldError bool
		expectedErr error
	}{
		{
			name: "returns_http_round_tripper",
			settings: &OAuth2ClientSettings{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError: false,
		},
		{
			name: "invalid_client_settings_should_error",
			settings: &OAuth2ClientSettings{
				ClientID: "testclientid",
				TokenURL: "https://example.com/v1/token",
				Scopes:   []string{"resource.read"},
			},
			shouldError: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			oauth2Authenticator, err := newOAuth2Extension(testcase.settings, zap.NewNop())
			if testcase.shouldError {
				assert.Error(t, err)
				assert.Nil(t, oauth2Authenticator)
				return
			}
			assert.NoError(t, err)
			perRPCCredentials, err := oauth2Authenticator.PerRPCCredentials()
			assert.Nil(t, err)
			// test perRPCCredentials is an grpc OAuthTokenSource
			_, ok := perRPCCredentials.(grpcOAuth.TokenSource)
			assert.True(t, ok)
		})
	}
}

func TestOAuthExtensionStart(t *testing.T) {
	oAuthExtensionAuth, err := newOAuth2Extension(
		&OAuth2ClientSettings{
			ClientID:     "testclientid",
			ClientSecret: "testsecret",
			TokenURL:     "https://example.com/v1/token",
			Scopes:       []string{"resource.read"},
		}, nil)
	assert.Nil(t, err)
	assert.Nil(t, oAuthExtensionAuth.Start(context.Background(), nil))
}

func TestOAuthExtensionShutdown(t *testing.T) {
	oAuthExtensionAuth, err := newOAuth2Extension(
		&OAuth2ClientSettings{
			ClientID:     "testclientid",
			ClientSecret: "testsecret",
			TokenURL:     "https://example.com/v1/token",
			Scopes:       []string{"resource.read"},
		}, nil)
	assert.Nil(t, err)
	assert.Nil(t, oAuthExtensionAuth.Shutdown(context.Background()))
}
