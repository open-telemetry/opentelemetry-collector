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

package testbed

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCase defines a running test case.
type TestCase struct {
	t *testing.T

	// Directory where test case results and logs will be written.
	resultDir string

	// Agent config file path.
	agentConfigFile string

	// Load generator spec file path.
	// loadSpecFile string

	// Resource spec for agent.
	resourceSpec resourceSpec

	// Agent process.
	agentProc childProcess

	LoadGenerator *LoadGenerator
	MockBackend   *MockBackend

	startTime time.Time

	// ErrorSignal indicates an error in the test case execution, e.g. process execution
	// failure or exceeding resource consumption, etc. The actual error message is already
	// logged, this is only an indicator on which you can wait to be informed.
	ErrorSignal chan struct{}

	doneSignal chan struct{}
}

const mibibyte = 1024 * 1024

// NewTestCase creates a new TestCase. It expected agent-config.yaml in the specified directory.
func NewTestCase(t *testing.T) *TestCase {
	tc := TestCase{}

	tc.t = t
	tc.ErrorSignal = make(chan struct{})
	tc.doneSignal = make(chan struct{})
	tc.startTime = time.Now()

	var err error
	tc.resultDir, err = filepath.Abs(path.Join("results", t.Name()))
	if err != nil {
		t.Fatalf("Cannot resolve %s: %v", t.Name(), err)
	}
	err = os.MkdirAll(tc.resultDir, os.ModePerm)
	if err != nil {
		t.Fatalf("Cannot create directory %s: %v", tc.resultDir, err)
	}

	// Set default resource check period.
	tc.resourceSpec.resourceCheckPeriod = 3 * time.Second

	// Locations of agent config and load generator spec files.
	tc.agentConfigFile, err = filepath.Abs(path.Join("testdata", "agent-config.yaml"))
	if err != nil {
		tc.t.Fatalf("Cannot resolve filename: %s", err.Error())
	}

	tc.LoadGenerator, err = NewLoadGenerator()
	if err != nil {
		t.Fatalf("Cannot create generator: %s", err.Error())
	}

	tc.MockBackend = NewMockBackend(tc.composeTestResultFileName("backend.log"))

	go tc.logStats()

	return &tc
}

func (tc *TestCase) composeTestResultFileName(fileName string) string {
	fileName, err := filepath.Abs(path.Join(tc.resultDir, fileName))
	if err != nil {
		tc.t.Fatalf("Cannot resolve %s: %s", fileName, err.Error())
	}
	return fileName
}

// SetExpectedMaxCPU sets the percentage of one core agent is expected to consume at most.
// Error is signalled if consumption during resourceCheckPeriod exceeds this number.
// If this function is not called the CPU consumption does not affect the test result.
func (tc *TestCase) SetExpectedMaxCPU(cpuPercentage uint32) {
	tc.resourceSpec.expectedMaxCPU = cpuPercentage
}

// SetExpectedMaxRAM sets the maximum RAM in MiB the agent is expected to consume.
// Error is signalled if consumption during resourceCheckPeriod exceeds this number.
// If this function is not called the RAM consumption does not affect the test result.
func (tc *TestCase) SetExpectedMaxRAM(ramMiB uint32) {
	tc.resourceSpec.expectedMaxRAM = ramMiB
}

// StartAgent starts the agent and redirects its standard output and standard error
// to "agent.log" file located in the test directory.
func (tc *TestCase) StartAgent() {
	logFileName := tc.composeTestResultFileName("agent.log")

	err := tc.agentProc.start(startParams{
		name:        "Agent",
		logFilePath: logFileName,
		cmd:         testBedConfig.Agent,
		cmdArgs:     []string{"--config", tc.agentConfigFile},
	})

	if err != nil {
		tc.indicateError(err)
		return
	}

	// Start watching if expected resources are defined.
	if tc.resourceSpec.expectedMaxCPU != 0 || tc.resourceSpec.expectedMaxRAM != 0 {
		go func() {
			err := tc.agentProc.watchResourceConsumption(&tc.resourceSpec)
			if err != nil {
				tc.indicateError(err)
			}
		}()
	}

	// Wait a bit for agent to start. This is a hack. We need to have a way to
	// wait for agent to start properly.
	time.Sleep(200 * time.Millisecond)
}

// StopAgent stops agent process.
func (tc *TestCase) StopAgent() {
	tc.agentProc.stop()
}

// StartLoad starts the load generator and redirects its standard output and standard error
// to "load-generator.log" file located in the test directory.
func (tc *TestCase) StartLoad(options LoadOptions) {
	tc.LoadGenerator.Start(options)
}

// StopLoad stops load generator.
func (tc *TestCase) StopLoad() {
	tc.LoadGenerator.Stop()
}

// StartBackend starts the specified backend type.
func (tc *TestCase) StartBackend(backendType BackendType) {
	if err := tc.MockBackend.Start(backendType); err != nil {
		tc.t.Fatalf("Cannot start backend: %s", err.Error())
	}
}

// StopBackend stops the backend.
func (tc *TestCase) StopBackend() {
	tc.MockBackend.Stop()
}

// Stop stops the load generator, the agent and the backend.
func (tc *TestCase) Stop() {
	// Stop all components
	tc.StopLoad()
	tc.StopAgent()
	tc.StopBackend()

	// Stop logging
	close(tc.doneSignal)

	// Report test results

	rc := tc.agentProc.GetTotalConsumption()

	var result string
	if tc.t.Failed() {
		result = "FAIL"
	} else {
		result = "PASS"
	}

	results.Add(tc.t.Name(), &TestResult{
		testName:          tc.t.Name(),
		result:            result,
		receivedSpanCount: tc.MockBackend.SpansReceived(),
		sentSpanCount:     tc.LoadGenerator.SpansSent,
		duration:          time.Since(tc.startTime),
		cpuPercentageAvg:  rc.CPUPercentAvg,
		cpuPercentageMax:  rc.CPUPercentMax,
		ramMibAvg:         rc.RAMMiBAvg,
		ramMibMax:         rc.RAMMiBMax,
	})
}

// ValidateData validates data by comparing the number of spans sent by load generator
// and number of spans received by mock backend.
func (tc *TestCase) ValidateData() {
	if assert.EqualValues(tc.t, tc.LoadGenerator.SpansSent, tc.MockBackend.SpansReceived(),
		"Received and sent span counters do not match.") {
		log.Printf("Sent and received data matches.")
	}
}

func (tc *TestCase) indicateError(err error) {
	// Print to log for visibility
	log.Print(err.Error())

	// Indicate error for the test
	tc.t.Error(err.Error())

	// Signal the error via channel
	close(tc.ErrorSignal)
}

func (tc *TestCase) logStats() {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			tc.logStatsOnce()
		case <-tc.doneSignal:
			return
		}
	}
}

func (tc *TestCase) logStatsOnce() {
	log.Printf("%s, %s, %s",
		tc.agentProc.GetResourceConsumption(),
		tc.LoadGenerator.GetStats(),
		tc.MockBackend.GetStats())
}
