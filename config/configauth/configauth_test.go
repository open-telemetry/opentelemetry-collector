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
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var mockID = component.MustNewID("mock")

func must[T any](t *testing.T, builder func() (T, error)) T {
	t.Helper()
	thing, err := builder()
	require.NoError(t, err)
	return thing
}

func TestGetServer(t *testing.T) {
	testCases := []struct {
		name          string
		authenticator extension.Extension
		expected      error
	}{
		{
			name: "obtain server authenticator",
			authenticator: must(t, func() (extension.Extension, error) {
				return extensionauth.NewServer()
			}),
			expected: nil,
		},
		{
			name: "not a server authenticator",
			authenticator: must(t, func() (extension.Extension, error) {
				return extensionauth.NewClient()
			}),
			expected: errNotServer,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Authentication{
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
	cfg := &Authentication{
		AuthenticatorID: component.MustNewID("does_not_exist"),
	}

	authenticator, err := cfg.GetServerAuthenticator(context.Background(), map[component.ID]component.Component{})
	require.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}

func TestGetClient(t *testing.T) {
	testCases := []struct {
		name          string
		authenticator extension.Extension
		expected      error
	}{
		{
			name: "obtain client authenticator",
			authenticator: must(t, func() (extension.Extension, error) {
				return extensionauth.NewClient()
			}),
			expected: nil,
		},
		{
			name: "not a client authenticator",
			authenticator: must(t, func() (extension.Extension, error) {
				return extensionauth.NewServer()
			}),
			expected: errNotClient,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Authentication{
				AuthenticatorID: mockID,
			}
			ext := map[component.ID]component.Component{
				mockID: tt.authenticator,
			}

			authenticator, err := cfg.GetClientAuthenticator(context.Background(), ext)

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

func TestGetClientFails(t *testing.T) {
	cfg := &Authentication{
		AuthenticatorID: component.MustNewID("does_not_exist"),
	}
	authenticator, err := cfg.GetClientAuthenticator(context.Background(), map[component.ID]component.Component{})
	require.ErrorIs(t, err, errAuthenticatorNotFound)
	assert.Nil(t, authenticator)
}
