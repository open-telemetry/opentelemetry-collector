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

package healthcheckextension

import (
	"net"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/extension/extensiontest"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

func TestHealthCheckExtensionUsage(t *testing.T) {
	config := Config{
		Port: testutils.GetAvailablePort(t),
	}

	hcExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, hcExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, hcExt.Start(mh))
	defer hcExt.Shutdown()

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	client := &http.Client{}
	url := "http://localhost:" + strconv.Itoa(int(config.Port))
	resp0, err := client.Get(url)
	require.NoError(t, err)
	defer resp0.Body.Close()

	require.Equal(t, http.StatusServiceUnavailable, resp0.StatusCode)

	hcExt.Ready()
	resp1, err := client.Get(url)
	require.NoError(t, err)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	hcExt.NotReady()
	resp2, err := client.Get(url)
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp2.StatusCode)
}

func TestHealthCheckExtensionPortAlreadyInUse(t *testing.T) {
	endpoint := testutils.GetAvailableLocalAddress(t)
	_, portStr, err := net.SplitHostPort(endpoint)
	require.NoError(t, err)

	// This needs to be ":port" because health checks also tries to connect to ":port".
	// To avoid the pop-up "accept incoming network connections" health check should be changed
	// to accept an address.
	ln, err := net.Listen("tcp", ":"+portStr)
	require.NoError(t, err)
	defer ln.Close()

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	config := Config{
		Port: uint16(port),
	}
	hcExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, hcExt)

	// Health check will report port already in use in a goroutine, use the mock
	// host to get it.
	mh := extensiontest.NewMockHost()
	require.NoError(t, hcExt.Start(mh))

	receivedError, receivedErr := mh.WaitForFatalError(500 * time.Millisecond)
	require.True(t, receivedError)
	require.Error(t, receivedErr)
}

func TestHealthCheckMultipleStarts(t *testing.T) {
	config := Config{
		Port: testutils.GetAvailablePort(t),
	}

	hcExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, hcExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, hcExt.Start(mh))
	defer hcExt.Shutdown()

	// Health check will report already in use in a goroutine, use the mock
	// host to get it.
	require.NoError(t, hcExt.Start(mh))

	receivedError, receivedErr := mh.WaitForFatalError(500 * time.Millisecond)
	require.True(t, receivedError)
	require.Error(t, receivedErr)
}

func TestHealthCheckMultipleShutdowns(t *testing.T) {
	config := Config{
		Port: testutils.GetAvailablePort(t),
	}

	hcExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, hcExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, hcExt.Start(mh))

	require.NoError(t, hcExt.Shutdown())
	require.NoError(t, hcExt.Shutdown())
}

func TestHealthCheckShutdownWithoutStart(t *testing.T) {
	config := Config{
		Port: testutils.GetAvailablePort(t),
	}

	hcExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, hcExt)

	require.NoError(t, hcExt.Shutdown())
}
