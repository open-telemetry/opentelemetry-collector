// Copyright 2019, OpenCensus Authors
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
	"testing"

	"github.com/census-instrumentation/opencensus-service/internal/zpagesserver"
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

	App.v.Set("config", "testdata/unisvc-config.yaml")

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
		addresses[i] = getAvailableLocalAddress(t)
	}
	return addresses
}

func getAvailableLocalAddress(t *testing.T) string {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get a free local port: %v", err)
	}
	// There is a possible race if something else takes this same port before
	// the test uses it, however, that is unlikely in practice.
	defer ln.Close()
	return ln.Addr().String()
}
