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

// +build windows

package pdh

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/third_party/telegraf/win_perf_counters"
)

type MockPerfCounter struct {
	returnValues []interface{}

	timesCalled int
	closed      bool
}

// NewMockPerfCounter creates a MockPerfCounter that returns the supplied
// values on each successive call to ScrapeData in the specified order.
// The supplied values must be of type float64 or error+
//
// If ScrapeData is called more times than the number of values supplied,
// the last supplied value will be returned for all subsequent calls.
func NewMockPerfCounter(valuesToBeReturned ...interface{}) *MockPerfCounter {
	return &MockPerfCounter{returnValues: valuesToBeReturned}
}

// Close
func (pc *MockPerfCounter) Close() error {
	if pc.closed {
		return errors.New("attempted to call close more than once")
	}

	pc.closed = true
	return nil
}

// ScrapeData
func (pc *MockPerfCounter) ScrapeData() ([]win_perf_counters.CounterValue, error) {
	returnIndex := pc.timesCalled
	if returnIndex >= len(pc.returnValues) {
		returnIndex = len(pc.returnValues) - 1
	}

	valueToReturn := pc.returnValues[returnIndex]

	pc.timesCalled++

	switch v := valueToReturn.(type) {
	case nil:
		return []win_perf_counters.CounterValue{{Value: 0}}, nil
	case float64:
		return []win_perf_counters.CounterValue{{Value: v}}, nil
	case error:
		return nil, v
	default:
		return nil, fmt.Errorf("supplied value to return was not of type float64 or error: %v (%T)", valueToReturn, valueToReturn)
	}
}
