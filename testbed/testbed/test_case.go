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

package testbed

import (
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCase defines a running test case.
type TestCase struct {
	t *testing.T

	// Directory where test case results and logs will be written.
	resultDir string

	// does not write out results when set to true
	skipResults bool

	// Agent config file path.
	agentConfigFile string

	// Load generator spec file path.
	// loadSpecFile string

	// Resource spec for agent.
	resourceSpec ResourceSpec

	// Agent process.
	agentProc OtelcolRunner

	Sender   DataSender
	receiver DataReceiver

	LoadGenerator *LoadGenerator
	MockBackend   *MockBackend
	validator     TestCaseValidator

	startTime time.Time

	// errorSignal indicates an error in the test case execution, e.g. process execution
	// failure or exceeding resource consumption, etc. The actual error message is already
	// logged, this is only an indicator on which you can wait to be informed.
	errorSignal chan struct{}
	// Duration is the requested duration of the tests. Configured via TESTBED_DURATION
	// env variable and defaults to 15 seconds if env variable is unspecified.
	Duration       time.Duration
	doneSignal     chan struct{}
	errorCause     string
	resultsSummary TestResultsSummary
}

const mibibyte = 1024 * 1024
const testcaseDurationVar = "TESTCASE_DURATION"

// NewTestCase creates a new TestCase. It expects agent-config.yaml in the specified directory.
func NewTestCase(
	t *testing.T,
	dataProvider DataProvider,
	sender DataSender,
	receiver DataReceiver,
	agentProc OtelcolRunner,
	validator TestCaseValidator,
	resultsSummary TestResultsSummary,
	opts ...TestCaseOption,
) *TestCase {
	tc := TestCase{
		t:              t,
		errorSignal:    make(chan struct{}),
		doneSignal:     make(chan struct{}),
		startTime:      time.Now(),
		Sender:         sender,
		receiver:       receiver,
		agentProc:      agentProc,
		validator:      validator,
		resultsSummary: resultsSummary,
	}

	// Get requested test case duration from env variable.
	duration := os.Getenv(testcaseDurationVar)
	if duration == "" {
		duration = "15s"
	}
	var err error
	tc.Duration, err = time.ParseDuration(duration)
	if err != nil {
		log.Fatalf("Invalid "+testcaseDurationVar+": %v. Expecting a valid duration string.", duration)
	}

	// Apply all provided options.
	for _, opt := range opts {
		opt(&tc)
	}

	// Prepare directory for results.
	tc.resultDir, err = filepath.Abs(path.Join("results", t.Name()))
	require.NoErrorf(t, err, "Cannot resolve %s", t.Name())
	require.NoErrorf(t, os.MkdirAll(tc.resultDir, os.ModePerm), "Cannot create directory %s", tc.resultDir)

	// Set default resource check period.
	tc.resourceSpec.ResourceCheckPeriod = 3 * time.Second
	if tc.Duration < tc.resourceSpec.ResourceCheckPeriod {
		// Resource check period should not be longer than entire test duration.
		tc.resourceSpec.ResourceCheckPeriod = tc.Duration
	}

	tc.LoadGenerator, err = NewLoadGenerator(dataProvider, sender)
	require.NoError(t, err, "Cannot create generator")

	tc.MockBackend = NewMockBackend(tc.composeTestResultFileName("backend.log"), receiver)

	go tc.logStats()

	return &tc
}

func (tc *TestCase) composeTestResultFileName(fileName string) string {
	fileName, err := filepath.Abs(path.Join(tc.resultDir, fileName))
	require.NoError(tc.t, err, "Cannot resolve %s", fileName)
	return fileName
}

// StartAgent starts the agent and redirects its standard output and standard error
// to "agent.log" file located in the test directory.
func (tc *TestCase) StartAgent(args ...string) {
	if tc.agentConfigFile != "" {
		args = append(args, "--config")
		args = append(args, tc.agentConfigFile)
	}
	logFileName := tc.composeTestResultFileName("agent.log")

	err := tc.agentProc.Start(StartParams{
		Name:         "Agent",
		LogFilePath:  logFileName,
		CmdArgs:      args,
		resourceSpec: &tc.resourceSpec,
	})

	if err != nil {
		tc.indicateError(err)
		return
	}

	// Start watching resource consumption.
	go func() {
		err := tc.agentProc.WatchResourceConsumption()
		if err != nil {
			tc.indicateError(err)
		}
	}()

	endpoint := tc.LoadGenerator.sender.GetEndpoint()
	if endpoint != nil {
		// Wait for agent to start. We consider the agent started when we can
		// connect to the port to which we intend to send load. We only do this
		// if the endpoint is not-empty, i.e. the sender does use network (some senders
		// like text log writers don't).
		tc.WaitFor(func() bool {
			conn, err := net.Dial(tc.LoadGenerator.sender.GetEndpoint().Network(), tc.LoadGenerator.sender.GetEndpoint().String())
			if err == nil && conn != nil {
				conn.Close()
				return true
			}
			return false
		})
	}
}

// StopAgent stops agent process.
func (tc *TestCase) StopAgent() {
	if _, err := tc.agentProc.Stop(); err != nil {
		tc.indicateError(err)
	}
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
func (tc *TestCase) StartBackend() {
	require.NoError(tc.t, tc.MockBackend.Start(), "Cannot start backend")
}

// StopBackend stops the backend.
func (tc *TestCase) StopBackend() {
	tc.MockBackend.Stop()
}

// EnableRecording enables recording of all data received by MockBackend.
func (tc *TestCase) EnableRecording() {
	tc.MockBackend.EnableRecording()
}

// AgentMemoryInfo returns raw memory info struct about the agent
// as returned by github.com/shirou/gopsutil/process
func (tc *TestCase) AgentMemoryInfo() (uint32, uint32, error) {
	stat, err := tc.agentProc.GetProcessMon().MemoryInfo()
	if err != nil {
		return 0, 0, err
	}
	return uint32(stat.RSS / mibibyte), uint32(stat.VMS / mibibyte), nil
}

// Stop stops the load generator, the agent and the backend.
func (tc *TestCase) Stop() {
	// Stop all components
	tc.StopLoad()
	tc.StopAgent()
	tc.StopBackend()

	// Stop logging
	close(tc.doneSignal)

	if tc.skipResults {
		return
	}

	// Report test results
	tc.validator.RecordResults(tc)
}

// ValidateData validates data received by mock backend against what was generated and sent to the collector
// instance(s) under test by the LoadGenerator.
func (tc *TestCase) ValidateData() {
	select {
	case <-tc.errorSignal:
		// Error is already signaled and recorded. Validating data is pointless.
		return
	default:
	}

	tc.validator.Validate(tc)
}

// Sleep for specified duration or until error is signaled.
func (tc *TestCase) Sleep(d time.Duration) {
	select {
	case <-time.After(d):
	case <-tc.errorSignal:
	}
}

// WaitForN the specific condition for up to a specified duration. Records a test error
// if time is out and condition does not become true. If error is signaled
// while waiting the function will return false, but will not record additional
// test error (we assume that signaled error is already recorded in indicateError()).
func (tc *TestCase) WaitForN(cond func() bool, duration time.Duration, errMsg ...interface{}) bool {
	startTime := time.Now()

	// Start with 5 ms waiting interval between condition re-evaluation.
	waitInterval := time.Millisecond * 5

	for {
		if cond() {
			return true
		}

		select {
		case <-time.After(waitInterval):
		case <-tc.errorSignal:
			return false
		}

		// Increase waiting interval exponentially up to 500 ms.
		if waitInterval < time.Millisecond*500 {
			waitInterval *= 2
		}

		if time.Since(startTime) > duration {
			// Waited too long
			tc.t.Error("Time out waiting for", errMsg)
			return false
		}
	}
}

// WaitFor is like WaitForN but with a fixed duration of 10 seconds
func (tc *TestCase) WaitFor(cond func() bool, errMsg ...interface{}) bool {
	return tc.WaitForN(cond, time.Second*10, errMsg...)
}

func (tc *TestCase) indicateError(err error) {
	// Print to log for visibility
	log.Print(err.Error())

	// Indicate error for the test
	tc.t.Error(err.Error())

	tc.errorCause = err.Error()

	// Signal the error via channel
	close(tc.errorSignal)
}

func (tc *TestCase) logStats() {
	t := time.NewTicker(tc.resourceSpec.ResourceCheckPeriod)
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
	log.Printf("%s | %s | %s",
		tc.agentProc.GetResourceConsumption(),
		tc.LoadGenerator.GetStats(),
		tc.MockBackend.GetStats())
}
