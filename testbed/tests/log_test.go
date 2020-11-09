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

package tests

// This file contains Test functions which initiate the tests. The tests can be either
// coded in this file or use scenarios from perf_scenarios.go.

import (
	"testing"

	"go.opentelemetry.io/collector/testbed/testbed"
)

func TestLog10kDPS(t *testing.T) {
	flw := testbed.NewFluentBitFileLogWriter(testbed.DefaultHost, testbed.GetAvailablePort(t))
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
		extensions   map[string]string
	}{
		{
			name:     "OTLP",
			sender:   testbed.NewOTLPLogsDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 20,
				ExpectedMaxRAM: 70,
			},
		},
		{
			name:     "OTLP-HTTP",
			sender:   testbed.NewOTLPHTTPLogsDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			receiver: testbed.NewOTLPHTTPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 30,
				ExpectedMaxRAM: 70,
			},
		},
		{
			name:     "FluentBitToOTLP",
			sender:   flw,
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 150,
			},
			extensions: flw.Extensions(),
		},
	}

	processors := map[string]string{
		"batch": `
  batch:
`,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				performanceResultsSummary,
				processors,
				test.extensions,
			)
		})
	}
}
