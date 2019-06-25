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

// Package collector handles the command-line, configuration, and runs the OC collector.
package collector

import (
	"net"
	"net/http"
	"os"
	"runtime"
	"testing"

	stats "github.com/guillermo/go.procstat"

	"github.com/open-telemetry/opentelemetry-service/internal/testutils"

	"github.com/open-telemetry/opentelemetry-service/internal/zpagesserver"
)

func TestApplication_Start(t *testing.T) {
	App = newApp()

	portArg := []string{
		healthCheckHTTPPort, // Keep it as first since its address is used later.
		zpagesserver.ZPagesHTTPPort,
		"metrics-port",
		"receivers.opencensus.port",
	}
	addresses := getMultipleAvailableLocalAddresses(t, uint(len(portArg)))
	for i, addr := range addresses {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("failed to split host and port from %q: %v", addr, err)
		}
		App.v.Set(portArg[i], port)
	}

	// Without exporters the collector will start and just shutdown, no error is expected.
	App.v.Set("logging-exporter", true)

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		if err := App.Start(); err != nil {
			t.Fatalf("App.Start() got %v, want nil", err)
		}
	}()

	<-App.readyChan
	if !isAppAvailable(t, "http://"+addresses[0]) {
		t.Fatalf("App didn't reach ready state")
	}
	close(App.stopTestChan)
	<-appDone
}

func TestApplication_StartUnified(t *testing.T) {

	App = newApp()

	portArg := []string{
		healthCheckHTTPPort, // Keep it as first since its address is used later.
		zpagesserver.ZPagesHTTPPort,
		"metrics-port",
	}
	addresses := getMultipleAvailableLocalAddresses(t, uint(len(portArg)))
	for i, addr := range addresses {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("failed to split host and port from %q: %v", addr, err)
		}
		App.v.Set(portArg[i], port)
	}

	App.v.Set("config", "testdata/otelsvc-config.yaml")

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		if err := App.StartUnified(); err != nil {
			t.Fatalf("App.StartUnified() got %v, want nil", err)
		}
	}()

	<-App.readyChan

	if !isAppAvailable(t, "http://"+addresses[0]) {
		t.Fatalf("App didn't reach ready state")
	}
	close(App.stopTestChan)
	<-appDone
}

func testMemBallast(t *testing.T, app *Application, ballastSizeMiB int) {
	maxRssBytes := mibToBytes(50)
	minVirtualBytes := mibToBytes(ballastSizeMiB)

	portArg := []string{
		healthCheckHTTPPort, // Keep it as first since its address is used later.
		zpagesserver.ZPagesHTTPPort,
		"metrics-port",
		"receivers.opencensus.port",
	}

	addresses := getMultipleAvailableLocalAddresses(t, uint(len(portArg)))
	for i, addr := range addresses {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("failed to split host and port from %q: %v", addr, err)
		}
		app.v.Set(portArg[i], port)
	}

	// Without exporters the collector will start and just shutdown, no error is expected.
	app.v.Set("logging-exporter", true)
	app.v.Set("mem-ballast-size-mib", ballastSizeMiB)

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		if err := app.Start(); err != nil {
			t.Fatalf("app.Start() got %v, want nil", err)
		}
	}()

	<-app.readyChan
	if !isAppAvailable(t, "http://"+addresses[0]) {
		t.Fatalf("app didn't reach ready state")
	}
	stats := stats.Stat{Pid: os.Getpid()}
	err := stats.Update()
	if err != nil {
		panic(err)
	}

	if stats.Vsize < minVirtualBytes {
		t.Errorf("unexpected virtual memory size. expected: >=%d, got: %d", minVirtualBytes, stats.Vsize)
	}

	if stats.Rss > maxRssBytes {
		t.Errorf("unexpected RSS size. expected: <%d, got: %d", maxRssBytes, stats.Rss)
	}

	close(app.stopTestChan)
	<-appDone
}

// TestApplication_MemBallast starts a new instance of collector with different
// mem ballast sizes and ensures that ballast consumes virtual memory but does
// not count towards RSS mem
func TestApplication_MemBallast(t *testing.T) {
	cases := []int{0, 500, 1000}
	for i := 0; i < len(cases); i++ {
		runtime.GC()
		app := newApp()
		testMemBallast(t, app, cases[i])
	}
}

// isAppAvailable checks if the healthcheck server at the given endpoint is
// returning `available`.
func isAppAvailable(t *testing.T, healthCheckEndPoint string) bool {
	client := &http.Client{}
	resp, err := client.Get(healthCheckEndPoint)
	if err != nil {
		t.Fatalf("failed to get a response from health probe: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusNoContent
}

func getMultipleAvailableLocalAddresses(t *testing.T, numAddresses uint) []string {
	addresses := make([]string, numAddresses, numAddresses)
	for i := uint(0); i < numAddresses; i++ {
		addresses[i] = testutils.GetAvailableLocalAddress(t)
	}
	return addresses
}

func mibToBytes(mib int) uint64 {
	return uint64(mib) * 1024 * 1024
}
