// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

func TestServerCustomListener(t *testing.T) {
	id := component.MustNewID("listener")
	addr := confignet.AddrConfig{Transport: confignet.TransportTypeTCP, Endpoint: "127.0.0.1:0"}
	called := make(chan confignet.AddrConfig, 1)
	extensions := map[component.ID]component.Component{id: struct {
		extension.Extension
		extensionmiddleware.GetListenerFunc
	}{
		Extension: extensionmiddlewaretest.NewNop(),
		GetListenerFunc: func(context.Context) (extensionmiddleware.ListenContextFunc, error) {
			return func(ctx context.Context, network, address string) (net.Listener, error) {
				called <- confignet.AddrConfig{Transport: confignet.TransportType(network), Endpoint: address}
				return (&net.ListenConfig{}).Listen(ctx, network, address)
			}, nil
		},
	}}
	config := ServerConfig{
		NetAddr:  addr,
		Listener: configoptional.Some(configmiddleware.Config{ID: id}),
	}
	listener, err := config.ToListener(t.Context(), extensions)
	require.NoError(t, err)
	server, err := config.ToServer(t.Context(), extensions, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	ptraceotlp.RegisterGRPCServer(server, &grpcTraceServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	response, err := sendTestRequest(t, ClientConfig{
		Endpoint: listener.Addr().String(),
		TLS:      configtls.ClientConfig{Insecure: true},
	})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, addr, <-called)
}

func TestServerCustomListenerErrors(t *testing.T) {
	id := component.MustNewID("listener")
	config := ServerConfig{
		NetAddr:  confignet.AddrConfig{Transport: confignet.TransportTypeTCP, Endpoint: "127.0.0.1:0"},
		Listener: configoptional.Some(configmiddleware.Config{ID: id}),
	}
	_, err := config.ToListener(t.Context())
	require.ErrorContains(t, err, "middleware not found")

	wantErr := errors.New("listen failed")
	extensions := map[component.ID]component.Component{id: struct {
		extension.Extension
		extensionmiddleware.GetListenerFunc
	}{
		Extension: extensionmiddlewaretest.NewNop(),
		GetListenerFunc: func(context.Context) (extensionmiddleware.ListenContextFunc, error) {
			return func(context.Context, string, string) (net.Listener, error) { return nil, wantErr }, nil
		},
	}}
	_, err = config.ToListener(t.Context(), extensions)
	require.ErrorIs(t, err, wantErr)

	extensions[id] = struct {
		extension.Extension
		extensionmiddleware.GetListenerFunc
	}{
		Extension: extensionmiddlewaretest.NewNop(),
		GetListenerFunc: func(context.Context) (extensionmiddleware.ListenContextFunc, error) {
			return func(context.Context, string, string) (net.Listener, error) { return nil, nil }, nil
		},
	}
	_, err = config.ToListener(t.Context(), extensions)
	require.ErrorContains(t, err, "nil listener")
}

func TestServerListenerConfigUnmarshal(t *testing.T) {
	var config ServerConfig
	err := confmap.NewFromStringMap(map[string]any{
		"endpoint":  "127.0.0.1:0",
		"transport": "tcp",
		"listener":  map[string]any{"id": "listener"},
	}).Unmarshal(&config)
	require.NoError(t, err)
	require.True(t, config.Listener.HasValue())
	require.Equal(t, component.MustNewID("listener"), config.Listener.Get().ID)
}
