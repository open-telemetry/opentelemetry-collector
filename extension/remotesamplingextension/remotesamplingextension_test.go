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

package remotesamplingextension

import (
	"net"
	"net/http"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/extension/extensiontest"
)

func TestPerformanceProfilerExtensionUsage(t *testing.T) {
	config := Config{
		Port: 5778,
		Addr: "0.0.0.0:14250",
	}

	remotesamplingExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, remotesamplingExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, remotesamplingExt.Start(mh))
	defer remotesamplingExt.Shutdown()

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	client := &http.Client{}
	resp, err := client.Get("http://localhost:" + strconv.Itoa(int(config.Port)) + "/debug/pprof")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPerformanceProfilerExtensionPortAlreadyInUse(t *testing.T) {
	config := Config{
		Port: 5778,
		Addr: "0.0.0.0:14250",
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(int(config.Port)))
	require.NoError(t, err)
	defer ln.Close()

	remotesamplingExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, remotesamplingExt)

	mh := extensiontest.NewMockHost()
	require.Error(t, remotesamplingExt.Start(mh))
}

func TestPerformanceProfilerMultipleStarts(t *testing.T) {
	config := Config{
		Port: 5778,
		Addr: "0.0.0.0:14250",
	}

	remotesamplingExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, remotesamplingExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, remotesamplingExt.Start(mh))
	defer remotesamplingExt.Shutdown()

	// Try to start it again, it will fail since it is on the same endpoint.
	require.Error(t, remotesamplingExt.Start(mh))
}

func TestPerformanceProfilerMultipleShutdowns(t *testing.T) {
	config := Config{
		Port: 5778,
		Addr: "0.0.0.0:14250",
	}

	remotesamplingExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, remotesamplingExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, remotesamplingExt.Start(mh))

	require.NoError(t, remotesamplingExt.Shutdown())
	require.NoError(t, remotesamplingExt.Shutdown())
}

func TestPerformanceProfilerShutdownWithoutStart(t *testing.T) {
	config := Config{
		Port: 5778,
		Addr: "0.0.0.0:14250",
	}

	remotesamplingExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, remotesamplingExt)

	require.NoError(t, remotesamplingExt.Shutdown())
}
