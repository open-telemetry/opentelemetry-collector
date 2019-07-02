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
	"testing"
	"time"
	"strconv"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/testbed/testbed"
)

func TestBallastMemory(t *testing.T) {
	tests := []struct{
		ballastSize uint32
		maxRSS uint32
	}{
		{100, 50},
		{500, 70},
		{1000, 100},
	}

	for _, test := range tests {
		tc := testbed.NewTestCase(t, testbed.WithSkipResults())
		tc.SetExpectedMaxRAM(test.maxRSS)

		tc.StartAgent("--mem-ballast-size-mib", strconv.Itoa(int(test.ballastSize)))

		var rss, vms uint32
		// It is possible that the process is not ready or the ballast code path 
		// is not hit immediately so we give the process up to a couple of seconds
		// to fire up and setup ballast. 2 seconds is a long time for this case but
		// it is short enough to not be annoying if the test fails repeatedly
		tc.WaitForN(func() bool {
			rss, vms, _ = tc.AgentMemoryInfo()
			return vms > test.ballastSize
		}, time.Second * 2, "VMS must be greater than %d", test.ballastSize)

		assert.True(t, rss <= test.maxRSS, fmt.Sprintf("RSS must be less than or equal to %d", test.maxRSS))
		tc.Stop()
	}
}