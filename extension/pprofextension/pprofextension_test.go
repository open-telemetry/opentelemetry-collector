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

package pprofextension

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/testutil"
)

func TestPerformanceProfilerExtensionUsage(t *testing.T) {
	config := Config{
		Endpoint:             testutil.GetAvailableLocalAddress(t),
		BlockProfileFraction: 3,
		MutexProfileFraction: 5,
	}

	pprofExt := newServer(config, zap.NewNop())
	require.NotNil(t, pprofExt)

	require.NoError(t, pprofExt.Start(context.Background(), componenttest.NewNopHost()))
	defer pprofExt.Shutdown(context.Background())

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	_, pprofPort, err := net.SplitHostPort(config.Endpoint)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Get("http://localhost:" + pprofPort + "/debug/pprof")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPerformanceProfilerExtensionPortAlreadyInUse(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", endpoint)
	require.NoError(t, err)
	defer ln.Close()

	config := Config{
		Endpoint: endpoint,
	}
	pprofExt := newServer(config, zap.NewNop())
	require.NotNil(t, pprofExt)

	require.Error(t, pprofExt.Start(context.Background(), componenttest.NewNopHost()))
}

func TestPerformanceProfilerMultipleStarts(t *testing.T) {
	config := Config{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}

	pprofExt := newServer(config, zap.NewNop())
	require.NotNil(t, pprofExt)

	require.NoError(t, pprofExt.Start(context.Background(), componenttest.NewNopHost()))
	defer pprofExt.Shutdown(context.Background())

	// Try to start it again, it will fail since it is on the same endpoint.
	require.Error(t, pprofExt.Start(context.Background(), componenttest.NewNopHost()))
}

func TestPerformanceProfilerMultipleShutdowns(t *testing.T) {
	config := Config{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}

	pprofExt := newServer(config, zap.NewNop())
	require.NotNil(t, pprofExt)

	require.NoError(t, pprofExt.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, pprofExt.Shutdown(context.Background()))
	require.NoError(t, pprofExt.Shutdown(context.Background()))
}

func TestPerformanceProfilerShutdownWithoutStart(t *testing.T) {
	config := Config{
		Endpoint: testutil.GetAvailableLocalAddress(t),
	}

	pprofExt := newServer(config, zap.NewNop())
	require.NotNil(t, pprofExt)

	require.NoError(t, pprofExt.Shutdown(context.Background()))
}
