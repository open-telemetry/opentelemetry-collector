// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

func TestRoundTripperFunc(t *testing.T) {
	var called bool
	var httpClient HTTPClient = ClientRoundTripperFunc(func(base http.RoundTripper) (http.RoundTripper, error) {
		called = true
		return base, nil
	})

	rt, err := httpClient.RoundTripper(http.DefaultTransport)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, http.DefaultTransport, rt)
}

type customPerRPCCredentials struct{}

var _ credentials.PerRPCCredentials = (*customPerRPCCredentials)(nil)

func (c *customPerRPCCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return nil, nil
}

func (c *customPerRPCCredentials) RequireTransportSecurity() bool {
	return true
}

func TestWithPerRPCCredentialsFunc(t *testing.T) {
	var called bool
	var grpcClient GRPCClient = ClientPerRPCCredentialsFunc(func() (credentials.PerRPCCredentials, error) {
		called = true
		return &customPerRPCCredentials{}, nil
	})

	creds, err := grpcClient.PerRPCCredentials()
	require.NoError(t, err)
	assert.True(t, called)
	assert.NotNil(t, creds)
}
