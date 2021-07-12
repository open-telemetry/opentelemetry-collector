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
		testbed.NewChildProcessCollector(),
		&testbed.PerfTestValidator{},
		performanceResultsSummary,
		testbed.WithResourceLimits(testbed.ResourceSpec{ExpectedMaxCPU: 4, ExpectedMaxRAM: 70}),
	)
	defer tc.Stop()

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
			testbed.NewChildProcessCollector(),
			&testbed.PerfTestValidator{},
			performanceResultsSummary,
			testbed.WithSkipResults(),
			testbed.WithResourceLimits(testbed.ResourceSpec{ExpectedMaxRAM: test.maxRSS}),
		)
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

		// https://github.com/open-telemetry/opentelemetry-collector/issues/3233
		// given that the maxRSS isn't an absolute maximum and that the actual maximum might be a bit off,
		// we give some room here instead of failing when the memory usage isn't that much higher than the max
		lenientMax := 1.1 * float32(test.maxRSS)
		assert.LessOrEqual(t, float32(rss), lenientMax)
		tc.Stop()
	}
}
