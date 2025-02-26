// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

func TestNilStartAndShutdown(t *testing.T) {
	// prepare
	m := &MockClient{}

	// test and verify
	origCtx := context.Background()

	err := m.Start(origCtx, nil)
	require.NoError(t, err)

	err = m.Shutdown(origCtx)
	assert.NoError(t, err)
}

type customRoundTripper struct{}

func (c *customRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
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

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			tripper, err := tt.clientAuth.RoundTripper(nil)
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, tripper)
			require.NoError(t, err)
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

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			credential, err := tt.clientAuth.PerRPCCredentials()
			if err != nil {
				return
			}
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, credential)
			require.NoError(t, err)
			// check if the resultant tripper is indeed the one provided
			_, ok := credential.(*customPerRPCCredentials)
			assert.True(t, ok)
		})
	}
}
