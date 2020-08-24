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

// Package testbed allows to easily set up a test that requires running the agent
// and a load generator, measure and define resource consumption expectations
// for the agent, fail tests automatically when expectations are exceeded.
//
// Each test case requires a agent configuration file and (optionally) load
// generator spec file. Test cases are defined as regular Go tests.
//
// Agent and load generator must be pre-built and their paths must be specified in
// test bed config file. The config file location must be provided in TESTBED_CONFIG
// env variable.
package testbed

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"

	"go.opentelemetry.io/collector/config"
)

// GlobalConfig defines test bed configuration.
type GlobalConfig struct {
	Agent         string
	LoadGenerator string `mapstructure:"load-generator"`
}

var testBedConfig = GlobalConfig{}

const testBedConfigEnvVarName = "TESTBED_CONFIG"

// ErrSkipTests indicates that the tests must be skipped.
var ErrSkipTests = errors.New("skip tests")

// LoadConfig loads test bed config.
func LoadConfig() error {
	// Get the test bed config file location from env variable.
	testBedConfigFile := os.Getenv(testBedConfigEnvVarName)
	if testBedConfigFile == "" {
		log.Printf(testBedConfigEnvVarName + " is not defined, skipping E2E tests.")
		return ErrSkipTests
	}

	testBedConfigFile, err := filepath.Abs(testBedConfigFile)
	if err != nil {
		log.Fatalf("Cannot resolve file name %q: %s",
			testBedConfigFile, err.Error())
	}

	testBedConfigDir := path.Dir(testBedConfigFile)

	// Use templates to expand some selected content on the config file.
	cfgTemplate, err := template.ParseFiles(testBedConfigFile)
	if err != nil {
		log.Fatalf("Template failed to parse config file %q: %s",
			testBedConfigFile, err.Error())
	}

	templateVars := struct {
		GOOS   string
		GOARCH string
	}{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
	var buf bytes.Buffer
	if err = cfgTemplate.Execute(&buf, templateVars); err != nil {
		log.Fatalf("Configuration template failed to run on file %q: %s",
			testBedConfigFile, err.Error())
	}

	// Read the config.
	v := config.NewViper()
	v.SetConfigType("yaml")
	if err = v.ReadConfig(bytes.NewBuffer(buf.Bytes())); err != nil {
		log.Fatalf("Cannot load test bed config from %q: %s",
			testBedConfigFile, err.Error())
	}

	if err = v.UnmarshalExact(&testBedConfig); err != nil {
		log.Fatalf("Cannot load test bed config from %q: %s",
			testBedConfigFile, err.Error())
	}

	// Convert relative paths to absolute.
	testBedConfig.Agent, err = filepath.Abs(path.Join(testBedConfigDir, testBedConfig.Agent))
	if err != nil {
		log.Fatalf("Cannot resolve file name %q: %s",
			testBedConfig.Agent, err.Error())
	}

	testBedConfig.LoadGenerator, err = filepath.Abs(path.Join(testBedConfig.LoadGenerator))
	if err != nil {
		log.Fatalf("Cannot resolve file name %q: %s",
			testBedConfig.LoadGenerator, err.Error())
	}

	return nil
}

func Start(resultsSummary TestResultsSummary) error {
	// Load the test bed config first.
	err := LoadConfig()

	if err != nil {
		if err == ErrSkipTests {
			// Let the caller know all tests must be skipped.
			return err
		}
		// Any other error while loading the config is fatal.
		log.Fatalf(err.Error())
	}

	dir, err := filepath.Abs("results")
	if err != nil {
		log.Fatalf(err.Error())
	}
	resultsSummary.Init(dir)

	return err
}

func SaveResults(resultsSummary TestResultsSummary) {
	resultsSummary.Save()
}

// DoTestMain is intended to be run from TestMain somewhere in the test suit.
// This enables the testbed.
func DoTestMain(m *testing.M, resultsSummary TestResultsSummary) {
	// Load the test bed config first.
	err := Start(resultsSummary)

	if err == ErrSkipTests {
		// Test bed config is not loaded because the tests are globally skipped.
		os.Exit(0)
	}

	res := m.Run()

	SaveResults(resultsSummary)

	// Now run all tests.
	os.Exit(res)
}
