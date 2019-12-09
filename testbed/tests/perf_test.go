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

// This file contains Test functions which initiate the tests. The tests can be either
// coded in this file or use scenarios from perf_scenarios.go.

import (
	"path"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector/testbed/testbed"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

// TestMain is used to initiate setup, execution and tear down of testbed.
func TestMain(m *testing.M) {
	testbed.DoTestMain(m)
}

func TestIdleMode(t *testing.T) {
	tc := testbed.NewTestCase(t, testbed.NewJaegerExporter(testbed.DefaultJaegerPort), &testbed.OCReceiver{})
	defer tc.Stop()

	tc.SetExpectedMaxCPU(4)
	tc.SetExpectedMaxRAM(50)

	tc.StartAgent()

	tc.Sleep(tc.Duration)
}

func Test10kSPS(t *testing.T) {
	tests := []struct {
		name     string
		receiver testbed.Receiver
	}{
		{"JaegerRx", testbed.NewJaegerReceiver(int(testutils.GetAvailablePort(t)))},
		{"OpenCensusRx", &testbed.OCReceiver{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Scenario10kSPS(t, testbed.NewJaegerExporter(int(testutils.GetAvailablePort(t))), test.receiver)
		})
	}
}

func TestNoBackend10kSPS(t *testing.T) {
	tc := testbed.NewTestCase(t, testbed.NewJaegerExporter(testbed.DefaultJaegerPort), &testbed.OCReceiver{})
	defer tc.Stop()

	tc.SetExpectedMaxCPU(200)
	tc.SetExpectedMaxRAM(200)

	tc.StartAgent()
	tc.StartLoad(testbed.LoadOptions{SpansPerSecond: 10000})

	tc.Sleep(tc.Duration)
}

func Test1kSPSWithAttrs(t *testing.T) {
	Scenario1kSPSWithAttrs(t, []string{}, []TestCase{
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

func TestBallast1kSPSWithAttrs(t *testing.T) {
	args := []string{"--mem-ballast-size-mib", "1000"}
	Scenario1kSPSWithAttrs(t, args, []TestCase{
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

func TestBallast1kSPSWithAttrsAddAttributes(t *testing.T) {
	args := []string{"--mem-ballast-size-mib", "1000"}
	Scenario1kSPSWithAttrs(
		t,
		args,
		[]TestCase{
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
		},
		testbed.WithConfigFile(path.Join("testdata", "add-attributes-config.yaml")),
	)
}
