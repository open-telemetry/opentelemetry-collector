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

package perfcounters

import (
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

// MockPerfCounterScraperError returns the supplied errors when Scrape, GetObject,
// or GetValues are called.

type MockPerfCounterScraperError struct {
	scrapeErr    error
	getObjectErr error
	getValuesErr error
}

func NewMockPerfCounterScraperError(scrapeErr, getObjectErr, getValuesErr error) *MockPerfCounterScraperError {
	return &MockPerfCounterScraperError{scrapeErr: scrapeErr, getObjectErr: getObjectErr, getValuesErr: getValuesErr}
}

func (p *MockPerfCounterScraperError) Initialize(objects ...string) error {
	return nil
}

func (p *MockPerfCounterScraperError) Scrape() (PerfDataCollection, error) {
	if p.scrapeErr != nil {
		return nil, p.scrapeErr
	}

	return mockPerfDataCollectionError{getObjectErr: p.getObjectErr, getValuesErr: p.getValuesErr}, nil
}

type mockPerfDataCollectionError struct {
	getObjectErr error
	getValuesErr error
}

func (p mockPerfDataCollectionError) GetObject(objectName string) (PerfDataObject, error) {
	if p.getObjectErr != nil {
		return nil, p.getObjectErr
	}

	return mockPerfDataObjectError{getValuesErr: p.getValuesErr}, nil
}

type mockPerfDataObjectError struct {
	getValuesErr error
}

func (obj mockPerfDataObjectError) Filter(includeFS, excludeFS filterset.FilterSet, includeTotal bool) {
}

func (obj mockPerfDataObjectError) GetValues(counterNames ...string) ([]*CounterValues, error) {
	return nil, obj.getValuesErr
}
