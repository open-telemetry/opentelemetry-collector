// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pprofextension

import (
	"net"
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/extension/extensiontest"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

func TestPerformanceProfilerExtensionUsage(t *testing.T) {
	config := Config{
		Endpoint:             testutils.GetAvailableLocalAddress(t),
		BlockProfileFraction: 3,
		MutexProfileFraction: 5,
	}

	pprofExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, pprofExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, pprofExt.Start(mh))
	defer pprofExt.Shutdown()

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
	endpoint := testutils.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", endpoint)
	require.NoError(t, err)
	defer ln.Close()

	config := Config{
		Endpoint: endpoint,
	}
	pprofExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, pprofExt)

	mh := extensiontest.NewMockHost()
	require.Error(t, pprofExt.Start(mh))
}

func TestPerformanceProfilerMultipleStarts(t *testing.T) {
	config := Config{
		Endpoint: testutils.GetAvailableLocalAddress(t),
	}

	pprofExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, pprofExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, pprofExt.Start(mh))
	defer pprofExt.Shutdown()

	// Try to start it again, it will fail since it is on the same endpoint.
	require.Error(t, pprofExt.Start(mh))
}

func TestPerformanceProfilerMultipleShutdowns(t *testing.T) {
	config := Config{
		Endpoint: testutils.GetAvailableLocalAddress(t),
	}

	pprofExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, pprofExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, pprofExt.Start(mh))

	require.NoError(t, pprofExt.Shutdown())
	require.NoError(t, pprofExt.Shutdown())
}

func TestPerformanceProfilerShutdownWithoutStart(t *testing.T) {
	config := Config{
		Endpoint: testutils.GetAvailableLocalAddress(t),
	}

	pprofExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, pprofExt)

	require.NoError(t, pprofExt.Shutdown())
}
