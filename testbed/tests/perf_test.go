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

// Package tests contains test cases. To run the tests go to tests directory and run:
// TESTBED_CONFIG=local.yaml go test -v

package tests

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-service/testbed/testbed"
)

func TestMain(m *testing.M) {
	// Load the test bed config first.
	err := testbed.Start()

	if err == testbed.ErrSkipTests {
		// Test bed config is not loaded because the tests are globally skipped.
		os.Exit(0)
	}

	res := m.Run()

	testbed.SaveResults()

	// Now run all tests.
	os.Exit(res)
}

func genRandByteString(len int) string {
	b := make([]byte, len)
	for i := range b {
		b[i] = byte(rand.Intn(128))
	}
	return string(b)
}

func TestIdleMode(t *testing.T) {
	tc := testbed.NewTestCase(t)
	defer tc.Stop()

	tc.SetExpectedMaxCPU(4)
	tc.SetExpectedMaxRAM(50)

	tc.StartAgent()

	tc.Sleep(10 * time.Second)
}

func Test10kSPS(t *testing.T) {
	tc := testbed.NewTestCase(t)
	defer tc.Stop()

	tc.SetExpectedMaxCPU(150)
	tc.SetExpectedMaxRAM(70)

	tc.StartBackend(testbed.BackendOC)
	tc.StartAgent()
	tc.StartLoad(testbed.LoadOptions{SpansPerSecond: 10000})

	tc.Sleep(15 * time.Second)

	tc.StopLoad()

	tc.WaitFor(func() bool { return tc.LoadGenerator.SpansSent() == tc.MockBackend.SpansReceived() },
		"all spans received")

	tc.StopAgent()

	tc.ValidateData()
}

func TestNoBackend10kSPS(t *testing.T) {
	tc := testbed.NewTestCase(t)
	defer tc.Stop()

	tc.SetExpectedMaxCPU(200)
	tc.SetExpectedMaxRAM(50)

	tc.StartAgent()
	tc.StartLoad(testbed.LoadOptions{SpansPerSecond: 10000})

	tc.Sleep(10 * time.Second)
}

type testCase struct {
	attrCount      int
	attrSizeByte   int
	expectedMaxCPU uint32
	expectedMaxRAM uint32
}

func test1000SPSWithAttributes(t *testing.T, args []string, tests []testCase) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%d*%dbytes", test.attrCount, test.attrSizeByte), func(t *testing.T) {

			tc := testbed.NewTestCase(t)
			defer tc.Stop()

			tc.SetExpectedMaxCPU(test.expectedMaxCPU)
			tc.SetExpectedMaxRAM(test.expectedMaxRAM)

			tc.StartBackend(testbed.BackendOC)
			tc.StartAgent(args...)

			options := testbed.LoadOptions{SpansPerSecond: 1000}
			options.Attributes = make(map[string]interface{})

			// Generate attributes.
			for i := 0; i < test.attrCount; i++ {
				attrName := genRandByteString(rand.Intn(199) + 1)
				options.Attributes[attrName] = genRandByteString(rand.Intn(test.attrSizeByte*2-1) + 1)
			}

			tc.StartLoad(options)
			tc.Sleep(10 * time.Second)
			tc.StopLoad()

			tc.WaitFor(func() bool { return tc.LoadGenerator.SpansSent() == tc.MockBackend.SpansReceived() },
				"all spans received")

			tc.StopAgent()

			tc.ValidateData()
		})
	}
}

func Test1000SPSWithAttributes(t *testing.T) {
	test1000SPSWithAttributes(t, []string{}, []testCase{
		// No attributes.
		{
			attrCount:      0,
			attrSizeByte:   0,
			expectedMaxCPU: 30,
			expectedMaxRAM: 100,
		},

		// We generate 10 attributes each with average key length of 100 bytes and
		// average value length of 50 bytes so total size of attributes values is
		// 15000 bytes.
		{
			attrCount:      100,
			attrSizeByte:   50,
			expectedMaxCPU: 120,
			expectedMaxRAM: 100,
		},

		// Approx 10 KiB attributes.
		{
			attrCount:      10,
			attrSizeByte:   1000,
			expectedMaxCPU: 100,
			expectedMaxRAM: 100,
		},

		// Approx 100 KiB attributes.
		{
			attrCount:      20,
			attrSizeByte:   5000,
			expectedMaxCPU: 250,
			expectedMaxRAM: 100,
		},
	})
}

func TestBallast1000SPSWithAttributes(t *testing.T) {
	args := []string{"--mem-ballast-size-mib", "1000"}
	test1000SPSWithAttributes(t, args, []testCase{
		// No attributes.
		{
			attrCount:      0,
			attrSizeByte:   0,
			expectedMaxCPU: 30,
			expectedMaxRAM: 2000,
		},
		{
			attrCount:      100,
			attrSizeByte:   50,
			expectedMaxCPU: 80,
			expectedMaxRAM: 2000,
		},
		{
			attrCount:      10,
			attrSizeByte:   1000,
			expectedMaxCPU: 80,
			expectedMaxRAM: 2000,
		},
		{
			attrCount:      20,
			attrSizeByte:   5000,
			expectedMaxCPU: 120,
			expectedMaxRAM: 2000,
		},
	})
}
