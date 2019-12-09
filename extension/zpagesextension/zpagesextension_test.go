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

package zpagesextension

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

func TestZPagesExtensionUsage(t *testing.T) {
	config := Config{
		Endpoint: testutils.GetAvailableLocalAddress(t),
	}

	zpagesExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, zpagesExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, zpagesExt.Start(mh))
	defer zpagesExt.Shutdown()

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	_, zpagesPort, err := net.SplitHostPort(config.Endpoint)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Get("http://localhost:" + zpagesPort + "/debug/tracez")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestZPagesExtensionPortAlreadyInUse(t *testing.T) {
	endpoint := testutils.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", endpoint)
	require.NoError(t, err)
	defer ln.Close()

	config := Config{
		Endpoint: endpoint,
	}
	zpagesExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, zpagesExt)

	mh := extensiontest.NewMockHost()
	require.Error(t, zpagesExt.Start(mh))
}

func TestZPagesMultipleStarts(t *testing.T) {
	config := Config{
		Endpoint: testutils.GetAvailableLocalAddress(t),
	}

	zpagesExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, zpagesExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, zpagesExt.Start(mh))
	defer zpagesExt.Shutdown()

	// Try to start it again, it will fail since it is on the same endpoint.
	require.Error(t, zpagesExt.Start(mh))
}

func TestZPagesMultipleShutdowns(t *testing.T) {
	config := Config{
		Endpoint: testutils.GetAvailableLocalAddress(t),
	}

	zpagesExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, zpagesExt)

	mh := extensiontest.NewMockHost()
	require.NoError(t, zpagesExt.Start(mh))

	require.NoError(t, zpagesExt.Shutdown())
	require.NoError(t, zpagesExt.Shutdown())
}

func TestZPagesShutdownWithoutStart(t *testing.T) {
	config := Config{
		Endpoint: testutils.GetAvailableLocalAddress(t),
	}

	zpagesExt, err := newServer(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, zpagesExt)

	require.NoError(t, zpagesExt.Shutdown())
}
