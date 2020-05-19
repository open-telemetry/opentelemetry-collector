// Copyright 2020, OpenTelemetry Authors
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

// +build windows

package pdh

import "go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/third_party/telegraf/win_perf_counters"

type MockPerfCounter struct {
	ReturnValues []float64

	timesCalled int
}

// NewMockPerfCounter creates a MockPerfCounter that returns the supplied
// values on each successive call to ScrapeData in the specified order.
//
// If ScrapeData is called more times than the number of values supplied,
// the last supplied value will be returned for all subsequent calls.
func NewMockPerfCounter(valuesToBeReturned ...float64) *MockPerfCounter {
	return &MockPerfCounter{ReturnValues: valuesToBeReturned}
}

// Close
func (pc *MockPerfCounter) Close() error {
	return nil
}

// ScrapeData
func (pc *MockPerfCounter) ScrapeData() ([]win_perf_counters.CounterValue, error) {
	returnIndex := pc.timesCalled
	if returnIndex >= len(pc.ReturnValues) {
		returnIndex = len(pc.ReturnValues) - 1
	}

	returnValue := pc.ReturnValues[returnIndex]

	pc.timesCalled++

	returnValueArray := []win_perf_counters.CounterValue{
		win_perf_counters.CounterValue{Value: returnValue},
	}

	return returnValueArray, nil
}
