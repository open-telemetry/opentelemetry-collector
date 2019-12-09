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

package tests

// This file defines parametrized test scenarios and makes them public so that they can be
// also used by tests in custom builds of Collector (e.g. Collector Contrib).

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector/testbed/testbed"
)

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
		fmt.Print(err)
		return ""
	}
	defer file.Close()

	_, err = file.WriteString(config)
	if err != nil {
		fmt.Print(err)
		return ""
	}

	return file.Name()
}

// Run 10k spans/sec test using specified exporter and receiver protocols.
func Scenario10kSPS(t *testing.T, exporter testbed.TraceExporter, receiver testbed.Receiver) {
	configFile := createConfigFile(exporter, receiver)
	defer os.Remove(configFile)

	if configFile == "" {
		t.Fatal("Cannot create config file")
	}

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

// TestCase for Scenario1kSPSWithAttrs func.
type TestCase struct {
	attrCount      int
	attrSizeByte   int
	expectedMaxCPU uint32
	expectedMaxRAM uint32
}

func genRandByteString(len int) string {
	b := make([]byte, len)
	for i := range b {
		b[i] = byte(rand.Intn(128))
	}
	return string(b)
}

// Scenario1kSPSWithAttrs runs a performance test at 1k sps with specified span attributes
// and test options.
func Scenario1kSPSWithAttrs(t *testing.T, args []string, tests []TestCase, opts ...testbed.TestCaseOption) {
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
