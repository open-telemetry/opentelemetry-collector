// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package authtest

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/credentials"
)

func TestNilStartAndShutdown(t *testing.T) {
	// prepare
	m := &MockClient{}

	// test and verify
	origCtx := context.Background()

	err := m.Start(origCtx, nil)
	assert.NoError(t, err)

	err = m.Shutdown(origCtx)
	assert.NoError(t, err)
}

type customRoundTripper struct{}

func (c *customRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestMockRoundTripper(t *testing.T) {
	testcases := []struct {
		name        string
		expectedErr bool
		clientAuth  MockClient
	}{
		{
			name:        "no_error",
			expectedErr: false,
			clientAuth: MockClient{
				ResultRoundTripper: &customRoundTripper{},
				MustError:          false,
			},
		},
		{
			name:        "error",
			expectedErr: true,
			clientAuth: MockClient{
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
		clientAuth  MockClient
	}{
		{
			name:        "no_error",
			expectedErr: false,
			clientAuth: MockClient{
				ResultPerRPCCredentials: &customPerRPCCredentials{},
				MustError:               false,
			},
		},
		{
			name:        "error",
			expectedErr: true,
			clientAuth: MockClient{
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
