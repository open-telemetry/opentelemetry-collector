// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/internal/testutil"
)

type zpagesHost struct {
	component.Host
}

func newZPagesHost() *zpagesHost {
	return &zpagesHost{Host: componenttest.NewNopHost()}
}

func (*zpagesHost) RegisterZPages(*http.ServeMux, string) {}

var (
	_ registerableTracerProvider = (*registerableProvider)(nil)
	_ registerableTracerProvider = sdktrace.NewTracerProvider()
)

type registerableProvider struct {
	trace.TracerProvider
}

func (*registerableProvider) RegisterSpanProcessor(sdktrace.SpanProcessor)   {}
func (*registerableProvider) UnregisterSpanProcessor(sdktrace.SpanProcessor) {}

func newZpagesTelemetrySettings() component.TelemetrySettings {
	set := componenttest.NewNopTelemetrySettings()
	set.TracerProvider = &registerableProvider{set.TracerProvider}
	return set
}

func TestZPagesExtensionUsage(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  addr,
				Transport: confignet.TransportTypeTCP,
			},
		},
	}

	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Start(context.Background(), newZPagesHost()))
	t.Cleanup(func() { require.NoError(t, zpagesExt.Shutdown(context.Background())) })

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	client := &http.Client{}
	resp, err := client.Get("http://" + addr + "/debug/tracez")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestZPagesExtensionBadAuthExtension(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  "localhost:0",
				Transport: confignet.TransportTypeTCP,
			},
			Auth: configoptional.Some(confighttp.AuthConfig{
				Config: configauth.Config{
					AuthenticatorID: component.MustNewIDWithName("foo", "bar"),
				},
			}),
		},
	}
	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.EqualError(t, zpagesExt.Start(context.Background(), componenttest.NewNopHost()), `failed to resolve authenticator "foo/bar": authenticator not found`)
}

func TestZPagesExtensionPortAlreadyInUse(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", endpoint)
	require.NoError(t, err)
	defer ln.Close()

	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  endpoint,
				Transport: confignet.TransportTypeTCP,
			},
		},
	}
	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.Error(t, zpagesExt.Start(context.Background(), componenttest.NewNopHost()))
}

func TestZPagesMultipleStarts(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  testutil.GetAvailableLocalAddress(t),
				Transport: confignet.TransportTypeTCP,
			},
		},
	}

	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, zpagesExt.Shutdown(context.Background())) })

	// Try to start it again, it will fail since it is on the same endpoint.
	require.Error(t, zpagesExt.Start(context.Background(), componenttest.NewNopHost()))
}

func TestZPagesMultipleShutdowns(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  testutil.GetAvailableLocalAddress(t),
				Transport: confignet.TransportTypeTCP,
			},
		},
	}

	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, zpagesExt.Shutdown(context.Background()))
	require.NoError(t, zpagesExt.Shutdown(context.Background()))
}

func TestZPagesShutdownWithoutStart(t *testing.T) {
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  testutil.GetAvailableLocalAddress(t),
				Transport: confignet.TransportTypeTCP,
			},
		},
	}

	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Shutdown(context.Background()))
}

func TestZPagesEnableExpvar(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	cfg := &Config{
		ServerConfig: confighttp.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  addr,
				Transport: confignet.TransportTypeTCP,
			},
		},
		Expvar: ExpvarConfig{
			Enabled: true,
		},
	}

	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Start(context.Background(), newZPagesHost()))
	t.Cleanup(func() { require.NoError(t, zpagesExt.Shutdown(context.Background())) })

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	client := &http.Client{}
	resp, err := client.Get("http://" + addr + "/debug/expvarz")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
