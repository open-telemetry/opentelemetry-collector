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

// Package tests contains test cases. To run the tests go to tests directory and run:
// RUN_TESTBED=1 go test -v

package tests

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/testbed/testbed"
)

func TestIdleMode(t *testing.T) {
	options := testbed.LoadOptions{DataItemsPerSecond: 10_000, ItemsPerBatch: 10}
	dataProvider := testbed.NewPerfTestDataProvider(options)
	tc := testbed.NewTestCase(
		t,
		dataProvider,
		testbed.NewJaegerGRPCDataSender(testbed.DefaultHost, testbed.DefaultJaegerPort),
		testbed.NewOCDataReceiver(testbed.DefaultOCPort),
		&testbed.ChildProcess{},
		&testbed.PerfTestValidator{},
		performanceResultsSummary,
	)
	defer tc.Stop()

	tc.SetResourceLimits(testbed.ResourceSpec{ExpectedMaxCPU: 4, ExpectedMaxRAM: 55})
	tc.StartAgent()

	tc.Sleep(tc.Duration)
}

func TestBallastMemory(t *testing.T) {
	tests := []struct {
		ballastSize uint32
		maxRSS      uint32
	}{
		{100, 60},
		{500, 70},
		{1000, 100},
	}

	options := testbed.LoadOptions{DataItemsPerSecond: 10_000, ItemsPerBatch: 10}
	dataProvider := testbed.NewPerfTestDataProvider(options)
	for _, test := range tests {
		tc := testbed.NewTestCase(
			t,
			dataProvider,
			testbed.NewJaegerGRPCDataSender(testbed.DefaultHost, testbed.DefaultJaegerPort),
			testbed.NewOCDataReceiver(testbed.DefaultOCPort),
			&testbed.ChildProcess{},
			&testbed.PerfTestValidator{},
			performanceResultsSummary,
			testbed.WithSkipResults(),
		)
		tc.SetResourceLimits(testbed.ResourceSpec{ExpectedMaxRAM: test.maxRSS})

		tc.StartAgent("--mem-ballast-size-mib", strconv.Itoa(int(test.ballastSize)))

		var rss, vms uint32
		// It is possible that the process is not ready or the ballast code path
		// is not hit immediately so we give the process up to a couple of seconds
		// to fire up and setup ballast. 2 seconds is a long time for this case but
		// it is short enough to not be annoying if the test fails repeatedly
		tc.WaitForN(func() bool {
			rss, vms, _ = tc.AgentMemoryInfo()
			return vms > test.ballastSize
		}, time.Second*2, "VMS must be greater than %d", test.ballastSize)

		assert.True(t, rss <= test.maxRSS, fmt.Sprintf("RSS must be less than or equal to %d", test.maxRSS))
		tc.Stop()
	}
}
