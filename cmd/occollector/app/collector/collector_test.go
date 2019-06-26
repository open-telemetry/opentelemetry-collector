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

	"github.com/shirou/gopsutil/process"

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

func testMemBallast(t *testing.T, app *Application, ballastSizeMiB int, maxRSSBytes uint64) {
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

	vms, rss := getMemStats(t)

	if vms < minVirtualBytes {
		t.Errorf("unexpected virtual memory size. expected: >=%d, got: %d", minVirtualBytes, vms)
	}

	if rss > maxRSSBytes {
		t.Errorf("unexpected RSS size. expected: <%d, got: %d", maxRSSBytes, rss)
	}

	close(app.stopTestChan)
	<-appDone
}

// TestApplication_MemBallast starts a new instance of collector with different
// mem ballast sizes and ensures that ballast consumes virtual memory but does
// not count towards RSS mem
func TestApplication_MemBallast(t *testing.T) {
	supportedPlatforms := map[string]bool{
		"linux/386":     true,
		"linux/amd64":   true,
		"linux/arm":     true,
		"freebsd/amd64": true,
		"freebsd/386":   true,
		"freebsd/arm":   true,
		"darwin/amd64":  true,
		"darwin/386":    true,
		"openbsd/amd64": true,
		"windows/amd64": true,
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if _, ok := supportedPlatforms[platform]; !ok {
		t.Skipf("This test cannot run on the platform: %s", platform)
	}

	perInstanceRSS := mibToBytes(50)
	ballastCostPerMB := 0.045
	cases := []int{0, 500, 1000}
	for i := 0; i < len(cases); i++ {
		_, currentRSS := getMemStats(t)
		ballastCost := float64(cases[i]) * ballastCostPerMB
		maxRSS := currentRSS + perInstanceRSS + mibToBytes(int(ballastCost))
		app := newApp()
		testMemBallast(t, app, cases[i], maxRSS)
		runtime.GC()
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

func getMemStats(t *testing.T) (uint64, uint64) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		t.Fatalf("error reading process stats:%v ", err)
	}
	mem, err := proc.MemoryInfo()
	if err != nil {
		t.Fatalf("error reading process memory stats:%v ", err)
	}

	return mem.VMS, mem.RSS
}
