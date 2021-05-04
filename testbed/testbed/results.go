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
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
)

// TestResultsSummary defines the interface to record results of one category of testing.
type TestResultsSummary interface {
	// Init creates and open the file and write headers.
	Init(resultsDir string)
	// Add results for one test.
	Add(testName string, result interface{})
	// Save the total results and close the file.
	Save()
}

// PerformanceResults implements the TestResultsSummary interface with fields suitable for reporting
// performance test results.
type PerformanceResults struct {
	resultsDir     string
	resultsFile    *os.File
	perTestResults []*PerformanceTestResult
	totalDuration  time.Duration
}

// PerformanceTestResult reports the results of a single performance test.
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
	if err := os.MkdirAll(resultsDir, os.FileMode(0755)); err != nil {
		log.Fatal(err)
	}
	var err error
	r.resultsFile, err = os.Create(path.Join(r.resultsDir, "TESTRESULTS.md"))
	if err != nil {
		log.Fatal(err)
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
func (r *PerformanceResults) Add(_ string, result interface{}) {
	testResult, ok := result.(*PerformanceTestResult)
	if !ok {
		return
	}
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
	r.totalDuration += testResult.duration
}

// CorrectnessResults implements the TestResultsSummary interface with fields suitable for reporting data translation
// correctness test results.
type CorrectnessResults struct {
	resultsDir             string
	resultsFile            *os.File
	perTestResults         []*CorrectnessTestResult
	totalAssertionFailures uint64
	totalDuration          time.Duration
}

// CorrectnessTestResult reports the results of a single correctness test.
type CorrectnessTestResult struct {
	testName                   string
	result                     string
	duration                   time.Duration
	sentSpanCount              uint64
	receivedSpanCount          uint64
	traceAssertionFailureCount uint64
	traceAssertionFailures     []*TraceAssertionFailure
}

type TraceAssertionFailure struct {
	typeName      string
	dataComboName string
	fieldPath     string
	expectedValue interface{}
	actualValue   interface{}
	sumCount      int
}

func (af TraceAssertionFailure) String() string {
	return fmt.Sprintf("%s/%s e=%#v a=%#v ", af.dataComboName, af.fieldPath, af.expectedValue, af.actualValue)
}

func (r *CorrectnessResults) Init(resultsDir string) {
	r.resultsDir = resultsDir
	r.perTestResults = []*CorrectnessTestResult{}

	// Create resultsSummary file
	if err := os.MkdirAll(resultsDir, os.FileMode(0755)); err != nil {
		log.Fatal(err)
	}
	var err error
	r.resultsFile, err = os.Create(path.Join(r.resultsDir, "CORRECTNESSRESULTS.md"))
	if err != nil {
		log.Fatal(err)
	}

	// Write the header
	_, _ = io.WriteString(r.resultsFile,
		"# Test Results\n"+
			fmt.Sprintf("Started: %s\n\n", time.Now().Format(time.RFC1123Z))+
			"Test                                    |Result|Duration|Sent Items|Received Items|Failure Count|Failures\n"+
			"----------------------------------------|------|-------:|---------:|-------------:|------------:|--------\n")
}

func (r *CorrectnessResults) Add(_ string, result interface{}) {
	testResult, ok := result.(*CorrectnessTestResult)
	if !ok {
		return
	}
	consolidated := consolidateAssertionFailures(testResult.traceAssertionFailures)
	failuresStr := ""
	for _, af := range consolidated {
		failuresStr = fmt.Sprintf("%s%s,%#v!=%#v,count=%d; ", failuresStr, af.fieldPath, af.expectedValue,
			af.actualValue, af.sumCount)
	}
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("%-40s|%-6s|%7.0fs|%10d|%14d|%13d|%s\n",
			testResult.testName,
			testResult.result,
			testResult.duration.Seconds(),
			testResult.sentSpanCount,
			testResult.receivedSpanCount,
			testResult.traceAssertionFailureCount,
			failuresStr,
		),
	)
	r.perTestResults = append(r.perTestResults, testResult)
	r.totalAssertionFailures += testResult.traceAssertionFailureCount
	r.totalDuration += testResult.duration
}

func (r *CorrectnessResults) Save() {
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("\nTotal assertion failures: %d\n", r.totalAssertionFailures))
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("\nTotal duration: %.0fs\n", r.totalDuration.Seconds()))
	r.resultsFile.Close()
}

func consolidateAssertionFailures(failures []*TraceAssertionFailure) map[string]*TraceAssertionFailure {
	afMap := make(map[string]*TraceAssertionFailure)
	for _, f := range failures {
		summary := afMap[f.fieldPath]
		if summary == nil {
			summary = &TraceAssertionFailure{
				typeName:      f.typeName,
				dataComboName: f.dataComboName + "...",
				fieldPath:     f.fieldPath,
				expectedValue: f.expectedValue,
				actualValue:   f.actualValue,
			}
			afMap[f.fieldPath] = summary
		}
		summary.sumCount++
	}
	return afMap
}
