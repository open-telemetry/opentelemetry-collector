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

// Package receivertest define types and functions used to help test packages
// implementing the receiver package interfaces.
package receivertest

import (
	"context"

	"github.com/open-telemetry/opentelemetry-service/receiver"
)

// MockHost mocks a receiver.ReceiverHost for test purposes.
type MockHost struct {
	okToIngest bool
}

var _ receiver.Host = (*MockHost)(nil)

// Context returns a context provided by the host to be used on the receiver
// operations.
func (mh *MockHost) Context() context.Context {
	return context.Background()
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (mh *MockHost) ReportFatalError(err error) {
	// Do nothing for now.
}

// OkToIngest returns true when the receiver can inject the received data
// into the pipeline and false when it should drop the data and report
// error to the client.
func (mh *MockHost) OkToIngest() bool {
	return mh.okToIngest
}

// NewMockHost returns a new instance of MockHost with proper defaults for most
// tests.
func NewMockHost() receiver.Host {
	return &MockHost{
		okToIngest: true,
	}
}

// SetOkToIngest sets the value to be returned by OkToIngest method.
func (mh *MockHost) SetOkToIngest(value bool) {
	mh.okToIngest = value
}
