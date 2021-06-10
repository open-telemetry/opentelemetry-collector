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

package oauth2clientcredentialsauthextension

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	grpcOAuth "google.golang.org/grpc/credentials/oauth"

	"go.opentelemetry.io/collector/config/configtls"
)

func TestOAuthClientSettings(t *testing.T) {
	tests := []struct {
		name          string
		settings      *Config
		shouldError   bool
		expectedError string
	}{
		{
			name: "all_valid_settings",
			settings: &Config{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
				Timeout:      2,
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile:   "testdata/testCA.pem",
						CertFile: "testdata/test-cert.pem",
						KeyFile:  "testdata/test-key.pem",
					},
					Insecure:           false,
					InsecureSkipVerify: false,
				},
			},
			shouldError:   false,
			expectedError: "",
		},
		{
			name: "invalid_tls",
			settings: &Config{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
				Timeout:      2,
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile:   "testdata/testCA.pem",
						CertFile: "testdata/t/doestExist.pem",
						KeyFile:  "testdata/test-key.pem",
					},
					Insecure:           false,
					InsecureSkipVerify: false,
				},
			},
			shouldError:   true,
			expectedError: "failed to load TLS config: failed to load TLS cert and key",
		},
		{
			name: "missing_client_id",
			settings: &Config{
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError:   true,
			expectedError: errNoClientIDProvided.Error(),
		},
		{
			name: "missing_client_secret",
			settings: &Config{
				ClientID: "testclientid",
				TokenURL: "https://example.com/v1/token",
				Scopes:   []string{"resource.read"},
			},
			shouldError:   true,
			expectedError: errNoClientSecretProvided.Error(),
		},
		{
			name: "missing_token_url",
			settings: &Config{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				Scopes:       []string{"resource.read"},
			},
			shouldError:   true,
			expectedError: errNoTokenURLProvided.Error(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rc, err := newClientCredentialsExtension(test.settings, zap.NewNop())
			if test.shouldError {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), test.expectedError)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.settings.Scopes, rc.clientCredentials.Scopes)
			assert.Equal(t, test.settings.TokenURL, rc.clientCredentials.TokenURL)
			assert.Equal(t, test.settings.ClientSecret, rc.clientCredentials.ClientSecret)
			assert.Equal(t, test.settings.ClientID, rc.clientCredentials.ClientID)
			assert.Equal(t, test.settings.Timeout, rc.client.Timeout)

			// test tls settings
			transport := rc.client.Transport.(*http.Transport)
			tlsClientConfig := transport.TLSClientConfig
			tlsTestSettingConfig, err := test.settings.TLSSetting.LoadTLSConfig()
			assert.Nil(t, err)
			assert.Equal(t, tlsClientConfig.Certificates, tlsTestSettingConfig.Certificates)
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
		settings    *Config
		shouldError bool
	}{
		{
			name: "returns_http_round_tripper",
			settings: &Config{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError: false,
		},
		{
			name: "invalid_client_settings_should_error",
			settings: &Config{
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
			oauth2Authenticator, err := newClientCredentialsExtension(testcase.settings, zap.NewNop())
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
		settings    *Config
		shouldError bool
		expectedErr error
	}{
		{
			name: "returns_http_round_tripper",
			settings: &Config{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
				Timeout:      1,
			},
			shouldError: false,
		},
		{
			name: "invalid_client_settings_should_error",
			settings: &Config{
				ClientID: "testclientid",
				TokenURL: "https://example.com/v1/token",
				Scopes:   []string{"resource.read"},
			},
			shouldError: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			oauth2Authenticator, err := newClientCredentialsExtension(testcase.settings, zap.NewNop())
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
	oAuthExtensionAuth, err := newClientCredentialsExtension(
		&Config{
			ClientID:     "testclientid",
			ClientSecret: "testsecret",
			TokenURL:     "https://example.com/v1/token",
			Scopes:       []string{"resource.read"},
		}, nil)
	assert.Nil(t, err)
	assert.Nil(t, oAuthExtensionAuth.Start(context.Background(), nil))
}

func TestOAuthExtensionShutdown(t *testing.T) {
	oAuthExtensionAuth, err := newClientCredentialsExtension(
		&Config{
			ClientID:     "testclientid",
			ClientSecret: "testsecret",
			TokenURL:     "https://example.com/v1/token",
			Scopes:       []string{"resource.read"},
		}, nil)
	assert.Nil(t, err)
	assert.Nil(t, oAuthExtensionAuth.Shutdown(context.Background()))
}
