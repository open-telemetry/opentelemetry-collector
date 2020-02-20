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
package component

import (
	"errors"
	"testing"
)

func TestNewMockHost(t *testing.T) {
	got := NewMockHost()
	if got == nil {
		t.Fatal("NewMockHost() = nil, want non-nil", got)
	}
	if ctx := got.Context(); ctx == nil {
		t.Fatalf("Context() = nil, want non-nil")
	}
	_, ok := got.(*MockHost)
	if !ok {
		t.Fatal("got.(*MockHost) failed")
	}
	got.ReportFatalError(errors.New("TestError"))
}
