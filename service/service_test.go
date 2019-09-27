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
package service

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/defaults"
	"github.com/open-telemetry/opentelemetry-collector/internal/testutils"
)

func TestApplication_StartUnified(t *testing.T) {
	factories, err := defaults.Components()
	assert.Nil(t, err)

	app := New(factories)

	portArg := []string{
		"metrics-port",
	}
	addresses := getMultipleAvailableLocalAddresses(t, uint(len(portArg)))
	for i, addr := range addresses {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("failed to split host and port from %q: %v", addr, err)
		}
		app.v.Set(portArg[i], port)
	}

	app.v.Set("config", "testdata/otelcol-config.yaml")

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		if err := app.StartUnified(); err != nil {
			t.Errorf("app.StartUnified() got %v, want nil", err)
			return
		}
	}()

	<-app.readyChan

	// TODO: Add a way to change configuration files so we can get the ports dynamically
	if !isAppAvailable(t, "http://localhost:13133") {
		t.Fatalf("app didn't reach ready state")
	}

	close(app.stopTestChan)
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
	return resp.StatusCode == http.StatusOK
}

func getMultipleAvailableLocalAddresses(t *testing.T, numAddresses uint) []string {
	addresses := make([]string, numAddresses)
	for i := uint(0); i < numAddresses; i++ {
		addresses[i] = testutils.GetAvailableLocalAddress(t)
	}
	return addresses
}
