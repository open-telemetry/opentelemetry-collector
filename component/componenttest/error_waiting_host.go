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

package componenttest

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// ErrorWaitingHost mocks an component.Host for test purposes.
type ErrorWaitingHost struct {
	errorChan chan error
}

var _ component.Host = (*ErrorWaitingHost)(nil)

// NewErrorWaitingHost returns a new instance of ErrorWaitingHost with proper defaults for most
// tests.
func NewErrorWaitingHost() *ErrorWaitingHost {
	return &ErrorWaitingHost{
		errorChan: make(chan error, 1),
	}
}

// ReportFatalError is used to report to the host that the extension encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (ews *ErrorWaitingHost) ReportFatalError(err error) {
	ews.errorChan <- err
}

// WaitForFatalError waits the given amount of time until an error is reported via
// ReportFatalError. It returns the error, if any, and a bool to indicated if
// an error was received before the time out.
func (ews *ErrorWaitingHost) WaitForFatalError(timeout time.Duration) (receivedError bool, err error) {
	select {
	case err = <-ews.errorChan:
		receivedError = true
	case <-time.After(timeout):
	}

	return
}

// GetFactory of the specified kind. Returns the factory for a component type.
func (ews *ErrorWaitingHost) GetFactory(_ component.Kind, _ configmodels.Type) component.Factory {
	return nil
}

func (ews *ErrorWaitingHost) GetExtensions() map[configmodels.Extension]component.ServiceExtension {
	return nil
}

func (ews *ErrorWaitingHost) GetExporters() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
	return nil
}
