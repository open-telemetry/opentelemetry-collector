// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/internal/testutil"
)

type zpagesHost struct {
	component.Host
}

func newZPagesHost() *zpagesHost {
	return &zpagesHost{Host: componenttest.NewNopHost()}
}

func (*zpagesHost) RegisterZPages(_ *http.ServeMux, _ string) {}

var _ registerableTracerProvider = (*registerableProvider)(nil)
var _ registerableTracerProvider = sdktrace.NewTracerProvider()

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
	cfg := &Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: testutil.GetAvailableLocalAddress(t),
		},
	}

	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Start(context.Background(), newZPagesHost()))
	t.Cleanup(func() { require.NoError(t, zpagesExt.Shutdown(context.Background())) })

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	_, zpagesPort, err := net.SplitHostPort(cfg.TCPAddr.Endpoint)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Get("http://localhost:" + zpagesPort + "/debug/tracez")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestZPagesExtensionPortAlreadyInUse(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", endpoint)
	require.NoError(t, err)
	defer ln.Close()

	cfg := &Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: endpoint,
		},
	}
	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.Error(t, zpagesExt.Start(context.Background(), componenttest.NewNopHost()))
}

func TestZPagesMultipleStarts(t *testing.T) {
	cfg := &Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: testutil.GetAvailableLocalAddress(t),
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
		TCPAddr: confignet.TCPAddr{
			Endpoint: testutil.GetAvailableLocalAddress(t),
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
		TCPAddr: confignet.TCPAddr{
			Endpoint: testutil.GetAvailableLocalAddress(t),
		},
	}

	zpagesExt := newServer(cfg, newZpagesTelemetrySettings())
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Shutdown(context.Background()))
}
