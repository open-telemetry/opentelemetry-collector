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
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/credentials"
)

func TestNilStartAndShutdown(t *testing.T) {
	// prepare
	m := &MockClientAuthenticator{}

	// test and verify
	origCtx := context.Background()

	err := m.Start(origCtx, nil)
	assert.NoError(t, err)

	err = m.Shutdown(origCtx)
	assert.NoError(t, err)
}

type customRoundTripper struct{}

func (c *customRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestMockRoundTripper(t *testing.T) {
	testcases := []struct {
		name        string
		expectedErr bool
		clientAuth  MockClientAuthenticator
	}{
		{
			name:        "no_error",
			expectedErr: false,
			clientAuth: MockClientAuthenticator{
				ResultRoundTripper: &customRoundTripper{},
				MustError:          false,
			},
		},
		{
			name:        "error",
			expectedErr: true,
			clientAuth: MockClientAuthenticator{
				ResultRoundTripper: &customRoundTripper{},
				MustError:          true,
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			tripper, err := testcase.clientAuth.RoundTripper(nil)
			if testcase.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, tripper)
			assert.NoError(t, err)
			// check if the resultant tripper is indeed the one provided
			_, ok := tripper.(*customRoundTripper)
			assert.True(t, ok)
		})
	}
}

type customPerRPCCredentials struct{}

var _ credentials.PerRPCCredentials = (*customPerRPCCredentials)(nil)

func (c *customPerRPCCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return nil, nil
}

func (c *customPerRPCCredentials) RequireTransportSecurity() bool {
	return true
}

func TestMockPerRPCCredential(t *testing.T) {
	testcases := []struct {
		name        string
		expectedErr bool
		clientAuth  MockClientAuthenticator
	}{
		{
			name:        "no_error",
			expectedErr: false,
			clientAuth: MockClientAuthenticator{
				ResultPerRPCCredentials: &customPerRPCCredentials{},
				MustError:               false,
			},
		},
		{
			name:        "error",
			expectedErr: true,
			clientAuth: MockClientAuthenticator{
				ResultPerRPCCredentials: &customPerRPCCredentials{},
				MustError:               true,
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			credential, err := testcase.clientAuth.PerRPCCredentials()
			if err != nil {
				return
			}
			if testcase.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, credential)
			assert.NoError(t, err)
			// check if the resultant tripper is indeed the one provided
			_, ok := credential.(*customPerRPCCredentials)
			assert.True(t, ok)
		})
	}
}
