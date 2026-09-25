// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

func newTestDialerExtension(dial func(ctx context.Context, network, address string) (net.Conn, error)) component.Component {
	return struct {
		extension.Extension
		extensionmiddleware.GetDialerFunc
	}{
		Extension: extensionmiddlewaretest.NewNop(),
		GetDialerFunc: func(context.Context) (func(ctx context.Context, network, address string) (net.Conn, error), error) {
			return dial, nil
		},
	}
}

func TestToClientConnCustomDialer(t *testing.T) {
	traceServer := &grpcTraceServer{}
	server, addr := traceServer.startTestServer(t, configoptional.Some(ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:0",
			Transport: confignet.TransportTypeTCP,
		},
	}))
	defer server.Stop()

	var dialed bool
	dialerID := component.MustNewID("dialer")
	extensions := map[component.ID]component.Component{
		dialerID: newTestDialerExtension(func(ctx context.Context, network, address string) (net.Conn, error) {
			dialed = true
			return (&net.Dialer{}).DialContext(ctx, network, address)
		}),
	}

	resp, err := sendTestRequestWithExtensions(t, ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		Dialer: configoptional.Some(configmiddleware.Config{ID: dialerID}),
	}, extensions)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, dialed, "expected the custom dialer to be used")
}

func TestToClientConnCustomDialerErrors(t *testing.T) {
	tests := []struct {
		name       string
		extensions map[component.ID]component.Component
		dialer     configmiddleware.Config
		errText    string
	}{
		{
			name:       "dialer_not_found",
			extensions: map[component.ID]component.Component{},
			dialer:     configmiddleware.Config{ID: component.MustNewID("nonexistent")},
			errText:    "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_dialer_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): struct {
					extension.Extension
					extensionmiddleware.GetDialerFunc
				}{
					Extension: extensionmiddlewaretest.NewNop(),
					GetDialerFunc: func(context.Context) (func(ctx context.Context, network, address string) (net.Conn, error), error) {
						return nil, errors.New("dialer error")
					},
				},
			},
			dialer:  configmiddleware.Config{ID: component.MustNewID("errormw")},
			errText: "dialer error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
				Dialer: configoptional.Some(tc.dialer),
			}
			_, err := cfg.ToClientConn(context.Background(), tc.extensions, componenttest.NewNopTelemetrySettings())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}
