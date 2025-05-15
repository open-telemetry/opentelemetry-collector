// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"
)

var mockID = component.MustNewID("mock")

func TestGetServer(t *testing.T) {
	testCases := []struct {
		name          string
		authenticator extension.Extension
		expected      error
	}{
		{
			name:          "obtain server authenticator",
			authenticator: extensionauthtest.NewNopServer(),
			expected:      nil,
		},
		{
			name:          "not a server authenticator",
			authenticator: extensionauthtest.NewNopClient(),
			expected:      errNotServer,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Config{
				AuthenticatorID: mockID,
			}
			ext := map[component.ID]component.Component{
				mockID: tt.authenticator,
			}

			authenticator, err := cfg.GetServerAuthenticator(context.Background(), ext)

			// verify
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
				assert.Nil(t, authenticator)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, authenticator)
			}
		})
	}
}

func TestGetServerFails(t *testing.T) {
	cfg := &Config{
		AuthenticatorID: component.MustNewID("does_not_exist"),
	}

	authenticator, err := cfg.GetServerAuthenticator(context.Background(), map[component.ID]component.Component{})
	require.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}

func TestGetHTTPClient(t *testing.T) {
	testCases := []struct {
		name          string
		authenticator extension.Extension
		expected      error
	}{
		{
			name:          "obtain client authenticator",
			authenticator: extensionauthtest.NewNopClient(),
			expected:      nil,
		},
		{
			name:          "not a client authenticator",
			authenticator: extensionauthtest.NewNopServer(),
			expected:      errNotHTTPClient,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Config{
				AuthenticatorID: mockID,
			}
			ext := map[component.ID]component.Component{
				mockID: tt.authenticator,
			}

			authenticator, err := cfg.GetHTTPClientAuthenticator(context.Background(), ext)

			// verify
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
				assert.Nil(t, authenticator)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, authenticator)
			}
		})
	}
}

func TestGetGRPCClientFails(t *testing.T) {
	cfg := &Config{
		AuthenticatorID: component.MustNewID("does_not_exist"),
	}
	authenticator, err := cfg.GetGRPCClientAuthenticator(context.Background(), map[component.ID]component.Component{})
	require.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}
