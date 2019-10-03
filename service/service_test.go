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
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/defaults"
	"github.com/open-telemetry/opentelemetry-collector/internal/testutils"
)

func TestApplication_Start(t *testing.T) {
	factories, err := defaults.Components()
	assert.Nil(t, err)

	app := New(factories)

	metricsPort := testutils.GetAvailablePort(t)
	app.rootCmd.SetArgs([]string{
		"--config=testdata/otelcol-config.yaml",
		"--metrics-port=" + strconv.FormatUint(uint64(metricsPort), 10),
	})

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		if err := app.Start(); err != nil {
			t.Errorf("app.Start() got %v, want nil", err)
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
