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

package extensiontest

import (
	"context"
	"time"

	"github.com/open-telemetry/opentelemetry-collector/component"
)

// MockHost mocks an component.Host for test purposes.
type MockHost struct {
	errorChan chan error
}

var _ component.Host = (*MockHost)(nil)

// NewMockHost returns a new instance of MockHost with proper defaults for most
// tests.
func NewMockHost() *MockHost {
	return &MockHost{
		errorChan: make(chan error, 1),
	}
}

// ReportFatalError is used to report to the host that the extension encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (mh *MockHost) ReportFatalError(err error) {
	mh.errorChan <- err
}

// Context returns a context provided by the host to be used on the component
// operations.
func (mh *MockHost) Context() context.Context {
	return context.Background()
}

// WaitForFatalError waits the given amount of time until an error is reported via
// ReportFatalError. It returns the error, if any, and a bool to indicated if
// an error was received before the time out.
func (mh *MockHost) WaitForFatalError(timeout time.Duration) (receivedError bool, err error) {
	select {
	case err = <-mh.errorChan:
		receivedError = true
	case <-time.After(timeout):
	}

	return
}
