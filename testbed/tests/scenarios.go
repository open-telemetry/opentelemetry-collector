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

	"github.com/open-telemetry/opentelemetry-collector/testbed/testbed"
)

// createConfigFile creates a collector config file that corresponds to the
// exporter and receiver used in the test and returns the config file name.
func createConfigFile(exporter testbed.Exporter, receiver testbed.Receiver) string {
	// Create a config. Note that our exporter is used to generate a config for Collector's
	// receiver and our receiver is used to generate a config for Collector's exporter.
	// This is because our exporter sends to Collector's receiver and our receiver
	// receives from Collector's exporter.

	var format string
	if _, ok := exporter.(testbed.TraceExporter); ok {
		// This is a trace test. Create appropriate config template.
		format = `
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
`
	} else {
		// This is a metric test. Create appropriate config template.
		format = `
receivers:%v
exporters:%v

service:
  pipelines:
    METRICS:
      receivers: [%v]
      exporters: [%v]
`
	}

	// Put corresponding elements into the config template to generate the final config.
	config := fmt.Sprintf(
		format,
		exporter.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		exporter.ProtocolName(),
		receiver.ProtocolName(),
	)

	// Write the config string to a temporary file.
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

	// Return config file name.
	return file.Name()
}

// Run 10k data items/sec test using specified exporter and receiver protocols.
func Scenario10kItemsPerSecond(
	t *testing.T,
	exporter testbed.Exporter,
	receiver testbed.Receiver,
	loadOptions testbed.LoadOptions,
) {
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

	if loadOptions.DataItemsPerSecond == 0 {
		// Use 10k spans or metric data points per second by default.
		loadOptions.DataItemsPerSecond = 10000
	}
	tc.StartLoad(loadOptions)

	tc.Sleep(tc.Duration)

	tc.StopLoad()

	tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
		"all data items received")

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

			tc := testbed.NewTestCase(
				t,
				testbed.NewJaegerExporter(testbed.DefaultJaegerPort),
				testbed.NewOCReceiver(testbed.DefaultOCPort),
				opts...,
			)
			defer tc.Stop()

			tc.SetExpectedMaxCPU(test.expectedMaxCPU)
			tc.SetExpectedMaxRAM(test.expectedMaxRAM)

			tc.StartBackend()
			tc.StartAgent(args...)

			options := testbed.LoadOptions{DataItemsPerSecond: 1000}
			options.Attributes = make(map[string]string)

			// Generate attributes.
			for i := 0; i < test.attrCount; i++ {
				attrName := genRandByteString(rand.Intn(199) + 1)
				options.Attributes[attrName] = genRandByteString(rand.Intn(test.attrSizeByte*2-1) + 1)
			}

			tc.StartLoad(options)
			tc.Sleep(tc.Duration)
			tc.StopLoad()

			tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
				"all spans received")

			tc.StopAgent()

			tc.ValidateData()
		})
	}
}
