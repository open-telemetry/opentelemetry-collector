// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configauth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"
)

var mockID = component.MustNewID("mock")

func TestGetHTTPHandler(t *testing.T) {
	testCases := []struct {
		name       string
		extensions map[component.ID]component.Component
		expected   error
	}{
		{
			name:       "not found",
			extensions: map[component.ID]component.Component{},
			expected:   errAuthenticatorNotFound,
		},
		{
			name: "not a server authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopClient(),
			},
			expected: errNotServer,
		},
		{
			name: "obtain server authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopServer(),
			},
			expected: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Config{
				AuthenticatorID: mockID,
			}

			handler, err := cfg.GetHTTPHandler(context.Background(), tt.extensions, nopHandler{}, []string{})
			// verify
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestGetGRPCServerOptions(t *testing.T) {
	testCases := []struct {
		name       string
		extensions map[component.ID]component.Component
		expected   error
	}{
		{
			name:       "not found",
			extensions: map[component.ID]component.Component{},
			expected:   errAuthenticatorNotFound,
		},
		{
			name: "not a server authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopClient(),
			},
			expected: errNotServer,
		},
		{
			name: "obtain server authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopServer(),
			},
			expected: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Config{
				AuthenticatorID: mockID,
			}

			opts, err := cfg.GetGRPCServerOptions(context.Background(), tt.extensions)
			// verify
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
			} else {
				require.NoError(t, err)
				assert.Len(t, opts, 2)
			}
		})
	}
}

func TestGetHTTPRoundTripper(t *testing.T) {
	testCases := []struct {
		name       string
		extensions map[component.ID]component.Component
		expected   error
	}{
		{
			name:       "not found",
			extensions: map[component.ID]component.Component{},
			expected:   errAuthenticatorNotFound,
		},
		{
			name: "not a client authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopServer(),
			},
			expected: errNotHTTPClient,
		},
		{
			name: "obtain client authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopClient(),
			},
			expected: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Config{
				AuthenticatorID: mockID,
			}
			rt, err := cfg.GetHTTPRoundTripper(context.Background(), tt.extensions, nopRoundTripper{})
			// verify
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, rt)
			}
		})
	}
}

func TestGetGRPCDialOptions(t *testing.T) {
	testCases := []struct {
		name       string
		extensions map[component.ID]component.Component
		expected   error
	}{
		{
			name:       "not found",
			extensions: map[component.ID]component.Component{},
			expected:   errAuthenticatorNotFound,
		},
		{
			name: "not a client authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopServer(),
			},
			expected: errNotGRPCClient,
		},
		{
			name: "obtain client authenticator",
			extensions: map[component.ID]component.Component{
				mockID: extensionauthtest.NewNopClient(),
			},
			expected: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := &Config{
				AuthenticatorID: mockID,
			}
			opts, err := cfg.GetGRPCDialOptions(context.Background(), tt.extensions)
			// verify
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
			} else {
				require.NoError(t, err)
				assert.Len(t, opts, 1)
			}
		})
	}
}

type nopRoundTripper struct{}

func (nopRoundTripper) RoundTrip(*http.Request) (*http.Response, error) { return &http.Response{}, nil }

type nopHandler struct{}

func (nopHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}
