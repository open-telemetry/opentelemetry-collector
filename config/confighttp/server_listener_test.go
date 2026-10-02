// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

func TestServerCustomListener(t *testing.T) {
	id := component.MustNewID("listener")
	called := make(chan confignet.AddrConfig, 1)
	config := ServerConfig{
		NetAddr:  confignet.AddrConfig{Transport: confignet.TransportTypeTCP, Endpoint: "127.0.0.1:0"},
		Listener: configoptional.Some(configmiddleware.Config{ID: id}),
	}
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
	listener, err := config.ToListener(t.Context(), extensions)
	require.NoError(t, err)
	server := &http.Server{
		ReadHeaderTimeout: time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	}
	t.Cleanup(func() { require.NoError(t, server.Close()) })
	go func() { _ = server.Serve(listener) }()

	resp, err := http.Get("http://" + listener.Addr().String())
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
	require.Equal(t, config.NetAddr, <-called)
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
