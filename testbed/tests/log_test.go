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

	Scenario10kItemsPerSecond(
		t,
		flw,
		testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
		testbed.ResourceSpec{
			ExpectedMaxCPU: 50,
			ExpectedMaxRAM: 99,
		},
		performanceResultsSummary,
		nil,
		flw.Extensions(),
	)

}
