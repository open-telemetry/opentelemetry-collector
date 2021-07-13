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

package awsecshealthcheckextension

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confignet"
)

func TestHealthCheckExtensionUsage(t *testing.T) {
	testEndpoint := "localhost:13134"
	config := Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: testEndpoint,
		},
		Interval:           "5m",
		ExporterErrorLimit: 1,
	}

	hcExt := newServer(config, zap.NewNop())
	require.NotNil(t, hcExt)

	require.NoError(t, hcExt.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, hcExt.Shutdown(context.Background())) })

	// Give a chance for the server goroutine to run.
	runtime.Gosched()

	currentTime := time.Now()
	vd1 := &view.Data{
		View:  nil,
		Start: currentTime.Add(-2 * time.Minute),
		End:   currentTime,
		Rows:  nil,
	}
	vd2 := &view.Data{
		View:  nil,
		Start: currentTime.Add(-1 * time.Minute),
		End:   currentTime,
		Rows:  nil,
	}

	hcExt.exporter.exporterErrorQueue = append(hcExt.exporter.exporterErrorQueue, vd1)
	req1, err := http.NewRequest("GET", testEndpoint, nil)
	require.Nil(t, err)
	responseRecorder1 := httptest.NewRecorder()
	hcExt.handler().ServeHTTP(responseRecorder1, req1)
	require.Equal(t, http.StatusOK, responseRecorder1.Code)

	hcExt.exporter.exporterErrorQueue = append(hcExt.exporter.exporterErrorQueue, vd2)
	req2, err := http.NewRequest("GET", testEndpoint, nil)
	require.Nil(t, err)
	responseRecorder2 := httptest.NewRecorder()
	hcExt.handler().ServeHTTP(responseRecorder2, req2)
	require.Equal(t, http.StatusInternalServerError, responseRecorder2.Code)
}

func TestHealthCheckExtensionPortAlreadyInUse(t *testing.T) {
	endpoint := "localhost:13134"

	ln, err := net.Listen("tcp", endpoint)
	require.NoError(t, err)
	defer ln.Close()

	config := Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: "localhost:13134",
		},
		Interval:           "5m",
		ExporterErrorLimit: 5,
	}
	hcExt := newServer(config, zap.NewNop())
	require.NotNil(t, hcExt)

	mh := newAssertNoErrorHost(t)
	require.Error(t, hcExt.Start(context.Background(), mh))
}

func TestHealthCheckMultipleStarts(t *testing.T) {
	config := Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: "localhost:13134",
		},
		Interval:           "5m",
		ExporterErrorLimit: 5,
	}

	hcExt := newServer(config, zap.NewNop())
	require.NotNil(t, hcExt)

	mh := newAssertNoErrorHost(t)
	require.NoError(t, hcExt.Start(context.Background(), mh))
	t.Cleanup(func() { require.NoError(t, hcExt.Shutdown(context.Background())) })

	require.Error(t, hcExt.Start(context.Background(), mh))
}

func TestHealthCheckMultipleShutdowns(t *testing.T) {
	config := Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: "localhost:13134",
		},
		Interval:           "5m",
		ExporterErrorLimit: 5,
	}

	hcExt := newServer(config, zap.NewNop())
	require.NotNil(t, hcExt)

	require.NoError(t, hcExt.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, hcExt.Shutdown(context.Background()))
	require.NoError(t, hcExt.Shutdown(context.Background()))
}

func TestHealthCheckShutdownWithoutStart(t *testing.T) {
	config := Config{
		TCPAddr: confignet.TCPAddr{
			Endpoint: "localhost:13134",
		},
		Interval:           "5m",
		ExporterErrorLimit: 5,
	}

	hcExt := newServer(config, zap.NewNop())
	require.NotNil(t, hcExt)

	require.NoError(t, hcExt.Shutdown(context.Background()))
}

// assertNoErrorHost implements a component.Host that asserts that there were no errors.
type assertNoErrorHost struct {
	component.Host
	*testing.T
}

// newAssertNoErrorHost returns a new instance of assertNoErrorHost.
func newAssertNoErrorHost(t *testing.T) component.Host {
	return &assertNoErrorHost{
		Host: componenttest.NewNopHost(),
		T:    t,
	}
}
