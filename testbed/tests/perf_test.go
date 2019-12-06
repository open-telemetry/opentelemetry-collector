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
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector/internal/testutils"
	"github.com/open-telemetry/opentelemetry-collector/testbed/testbed"
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
	tc := testbed.NewTestCase(t, testbed.NewJaegerExporter(testbed.DefaultJaegerPort), &testbed.OCReceiver{})
	defer tc.Stop()

	tc.SetExpectedMaxCPU(4)
	tc.SetExpectedMaxRAM(50)

	tc.StartAgent()

	tc.Sleep(10 * time.Second)
}

// createConfigFile creates a collector config file that corresponds to the
// exporter and receiver used in the test and returns the config file name.
func createConfigFile(exporter testbed.TraceExporter, receiver testbed.Receiver) string {
	// Create a config. Note that our exporter is used to generate a config for Collector's
	// receiver and our receiver is used to generate a config for Collector's exporter.
	// This is because our exporter sends to Collector's receiver and our receiver
	// receives from Collector's exporter.
	config := fmt.Sprintf(`
receivers:%v
exporters:%v
processors:
  queued_retry:

service:
  pipelines:
    traces:
      receivers: [%v]
      processors: [queued_retry]
      exporters: [%v]
`, exporter.GenConfigYAMLStr(), receiver.GenConfigYAMLStr(), exporter.ProtocolName(), receiver.ProtocolName())

	file, err := ioutil.TempFile("", "agent*.yaml")
	if err != nil {
		log.Fatal(err)
		return ""
	}

	_, err = file.WriteString(config)
	if err != nil {
		fmt.Println(err)
		file.Close()
		return ""
	}

	return file.Name()
}

// Run 10k spans/sec test using specified exporter and receiver protocols.
func RunTest10kSPS(t *testing.T, exporter testbed.TraceExporter, receiver testbed.Receiver) {
	configFile := createConfigFile(exporter, receiver)
	defer os.Remove(configFile)

	tc := testbed.NewTestCase(t, exporter, receiver, testbed.WithConfigFile(configFile))
	defer tc.Stop()

	tc.SetExpectedMaxCPU(150)
	tc.SetExpectedMaxRAM(70)

	tc.StartBackend()
	tc.StartAgent()
	tc.StartLoad(testbed.LoadOptions{SpansPerSecond: 10000})

	tc.Sleep(15 * time.Second)

	tc.StopLoad()

	tc.WaitFor(func() bool { return tc.LoadGenerator.SpansSent() == tc.MockBackend.SpansReceived() },
		"all spans received")

	tc.StopAgent()

	tc.ValidateData()
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
			RunTest10kSPS(t, testbed.NewJaegerExporter(int(testutils.GetAvailablePort(t))), test.receiver)
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

	tc.Sleep(10 * time.Second)
}

type testCase struct {
	attrCount      int
	attrSizeByte   int
	expectedMaxCPU uint32
	expectedMaxRAM uint32
}

func test1kSPSWithAttrs(t *testing.T, args []string, tests []testCase, opts ...testbed.TestCaseOption) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%d*%dbytes", test.attrCount, test.attrSizeByte), func(t *testing.T) {

			tc := testbed.NewTestCase(t, testbed.NewJaegerExporter(testbed.DefaultJaegerPort), &testbed.OCReceiver{}, opts...)
			defer tc.Stop()

			tc.SetExpectedMaxCPU(test.expectedMaxCPU)
			tc.SetExpectedMaxRAM(test.expectedMaxRAM)

			tc.StartBackend()
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

func Test1kSPSWithAttrs(t *testing.T) {
	test1kSPSWithAttrs(t, []string{}, []testCase{
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
	test1kSPSWithAttrs(t, args, []testCase{
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
	test1kSPSWithAttrs(
		t,
		args,
		[]testCase{
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
