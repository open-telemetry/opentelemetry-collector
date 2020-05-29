// Copyright The OpenTelemetry Authors
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
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
)

// TestResultsSummary defines the interface to record results of one category of testing.
type TestResultsSummary interface {
	// Create and open the file and write headers.
	Init(resultsDir string)
	// Add results for one test.
	Add(testName string, result interface{})
	// Save the total results and close the file.
	Save()
}

//var resultsSummary = &PerformanceResults{}

type PerformanceResults struct {
	resultsDir     string
	resultsFile    *os.File
	perTestResults []*PerformanceTestResult
	totalDuration  time.Duration
}

type PerformanceTestResult struct {
	testName          string
	result            string
	duration          time.Duration
	cpuPercentageAvg  float64
	cpuPercentageMax  float64
	ramMibAvg         uint32
	ramMibMax         uint32
	sentSpanCount     uint64
	receivedSpanCount uint64
	errorCause        string
}

func (r *PerformanceResults) Init(resultsDir string) {
	r.resultsDir = resultsDir
	r.perTestResults = []*PerformanceTestResult{}

	// Create resultsSummary file
	var err error
	r.resultsFile, err = os.Create(path.Join(r.resultsDir, "TESTRESULTS.md"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Write the header
	_, _ = io.WriteString(r.resultsFile,
		"# Test PerformanceResults\n"+
			fmt.Sprintf("Started: %s\n\n", time.Now().Format(time.RFC1123Z))+
			"Test                                    |Result|Duration|CPU Avg%|CPU Max%|RAM Avg MiB|RAM Max MiB|Sent Items|Received Items|\n"+
			"----------------------------------------|------|-------:|-------:|-------:|----------:|----------:|---------:|-------------:|\n")
}

// Save the total results and close the file.
func (r *PerformanceResults) Save() {
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("\nTotal duration: %.0fs\n", r.totalDuration.Seconds()))
	r.resultsFile.Close()
}

// Add results for one test.
func (r *PerformanceResults) Add(testName string, result interface{}) {
	testResult := result.(*PerformanceTestResult)
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("%-40s|%-6s|%7.0fs|%8.1f|%8.1f|%11d|%11d|%10d|%14d|%s\n",
			testResult.testName,
			testResult.result,
			testResult.duration.Seconds(),
			testResult.cpuPercentageAvg,
			testResult.cpuPercentageMax,
			testResult.ramMibAvg,
			testResult.ramMibMax,
			testResult.sentSpanCount,
			testResult.receivedSpanCount,
			testResult.errorCause,
		),
	)
	r.totalDuration = r.totalDuration + testResult.duration
}

type CorrectnessResults struct {
	resultsDir             string
	resultsFile            *os.File
	perTestResults         []*CorrectnessTestResult
	totalAssertionFailures uint64
	totalDuration          time.Duration
}

type CorrectnessTestResult struct {
	testName              string
	result                string
	duration              time.Duration
	sentSpanCount         uint64
	receivedSpanCount     uint64
	assertionFailureCount uint64
	assertionFailures     []*AssertionFailure
}

type AssertionFailure struct {
	typeName      string
	dataComboName string
	fieldPath     string
	expectedValue interface{}
	actualValue   interface{}
}

func (r *CorrectnessResults) Init(resultsDir string) {
	r.resultsDir = resultsDir
	r.perTestResults = []*CorrectnessTestResult{}

	// Create resultsSummary file
	var err error
	r.resultsFile, err = os.Create(path.Join(r.resultsDir, "CORRECTNESSRESULTS.md"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Write the header
	_, _ = io.WriteString(r.resultsFile,
		"# Test Results\n"+
			fmt.Sprintf("Started: %s\n\n", time.Now().Format(time.RFC1123Z))+
			"Test                                    |Result|Duration|Sent Items|Received Items|Failure Count|Failures\n"+
			"----------------------------------------|------|-------:|---------:|-------------:|------------:|--------\n")
}

func (r *CorrectnessResults) Add(testName string, result interface{}) {
	testResult := result.(*CorrectnessTestResult)
	failuresStr := ""
	for _, af := range testResult.assertionFailures {
		failuresStr = fmt.Sprintf("%s; %s:%s:%s:%s:%s", failuresStr, af.typeName, af.dataComboName,
			af.fieldPath, af.expectedValue, af.actualValue)
	}
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("%-40s|%-6s|%7.0fs|%10d|%14d|%13d|%s\n",
			testResult.testName,
			testResult.result,
			testResult.duration.Seconds(),
			testResult.sentSpanCount,
			testResult.receivedSpanCount,
			testResult.assertionFailureCount,
			failuresStr,
		),
	)
	r.perTestResults = append(r.perTestResults, testResult)
	r.totalAssertionFailures = r.totalAssertionFailures + testResult.assertionFailureCount
	r.totalDuration = r.totalDuration + testResult.duration
}

func (r *CorrectnessResults) Save() {
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("\nTotal assertion failures: %d\n", r.totalAssertionFailures))
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("\nTotal duration: %.0fs\n", r.totalDuration.Seconds()))
	r.resultsFile.Close()
}
