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

type Results struct {
	resultsDir     string
	resultsFile    *os.File
	perTestResults []*TestResult
	totalDuration  time.Duration
}

var results = Results{}

type TestResult struct {
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

func (r *Results) Init(resultsDir string) {
	r.resultsDir = resultsDir
	r.perTestResults = []*TestResult{}

	// Create results file
	var err error
	r.resultsFile, err = os.Create(path.Join(r.resultsDir, "TESTRESULTS.md"))
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Write the header
	_, _ = io.WriteString(r.resultsFile,
		"# Test Results\n"+
			fmt.Sprintf("Started: %s\n\n", time.Now().Format(time.RFC1123Z))+
			"Test                                    |Result|Duration|CPU Avg%|CPU Max%|RAM Avg MiB|RAM Max MiB|Sent Items|Received Items|\n"+
			"----------------------------------------|------|-------:|-------:|-------:|----------:|----------:|---------:|-------------:|\n")
}

// Save the total results and close the file.
func (r *Results) Save() {
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("\nTotal duration: %.0fs\n", r.totalDuration.Seconds()))
	r.resultsFile.Close()
}

// Add results for one test.
func (r *Results) Add(testName string, result *TestResult) {
	_, _ = io.WriteString(r.resultsFile,
		fmt.Sprintf("%-40s|%-6s|%7.0fs|%8.1f|%8.1f|%11d|%11d|%10d|%14d|%s\n",
			result.testName,
			result.result,
			result.duration.Seconds(),
			result.cpuPercentageAvg,
			result.cpuPercentageMax,
			result.ramMibAvg,
			result.ramMibMax,
			result.sentSpanCount,
			result.receivedSpanCount,
			result.errorCause,
		),
	)
	r.totalDuration = r.totalDuration + result.duration
}
