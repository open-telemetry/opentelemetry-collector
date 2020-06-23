// Copyright 2020, OpenTelemetry Authors
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
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCaseValidator defines the interface for validating and reporting test results.
type TestCaseValidator interface {
	// Validate executes validation routines and test assertions.
	Validate(tc *TestCase)
	// RecordResults updates the TestResultsSummary for the test suite with results of a single test.
	RecordResults(tc *TestCase)
}

// PerfTestValidator implements TestCaseValidator for test suites using PerformanceResults for summarizing results.
type PerfTestValidator struct {
}

func (v *PerfTestValidator) Validate(tc *TestCase) {
	if assert.EqualValues(tc.t, tc.LoadGenerator.DataItemsSent(), tc.MockBackend.DataItemsReceived(),
		"Received and sent counters do not match.") {
		log.Printf("Sent and received data matches.")
	}
}

func (v *PerfTestValidator) RecordResults(tc *TestCase) {
	rc := tc.agentProc.GetTotalConsumption()

	var result string
	if tc.t.Failed() {
		result = "FAIL"
	} else {
		result = "PASS"
	}

	// Remove "Test" prefix from test name.
	testName := tc.t.Name()[4:]

	tc.resultsSummary.Add(tc.t.Name(), &PerformanceTestResult{
		testName:          testName,
		result:            result,
		receivedSpanCount: tc.MockBackend.DataItemsReceived(),
		sentSpanCount:     tc.LoadGenerator.DataItemsSent(),
		duration:          time.Since(tc.startTime),
		cpuPercentageAvg:  rc.CPUPercentAvg,
		cpuPercentageMax:  rc.CPUPercentMax,
		ramMibAvg:         rc.RAMMiBAvg,
		ramMibMax:         rc.RAMMiBMax,
		errorCause:        tc.errorCause,
	})
}
