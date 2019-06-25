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
			"Test|Result|Duration|CPU Avg%|CPU Max%|RAM Avg MiB|RAM Max MiB|Sent Spans|Received Spans\n"+
			"----|------|-------:|-------:|-------:|----------:|----------:|---------:|-------------:\n")
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
		fmt.Sprintf("%s|%s|%.0fs|%.1f|%.1f|%d|%d|%d|%d\n",
			result.testName,
			result.result,
			result.duration.Seconds(),
			result.cpuPercentageAvg,
			result.cpuPercentageMax,
			result.ramMibAvg,
			result.ramMibMax,
			result.sentSpanCount,
			result.receivedSpanCount,
		),
	)
	r.totalDuration = r.totalDuration + result.duration
}
