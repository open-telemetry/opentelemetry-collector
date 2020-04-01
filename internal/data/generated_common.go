// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by "internal/data_generator/main.go". DO NOT EDIT.
// To regenerate this file run "go run internal/data_generator/main.go".

package data

import (
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
)

// InstrumentationLibrary is a message representing the instrumentation library information.
//
// This is a reference type, if passsed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewInstrumentationLibrary function to create new instances.
// Important: zero-initialized instance is not valid for use.
type InstrumentationLibrary struct {
	// Wrap OTLP otlpcommon.InstrumentationLibrary.
	orig **otlpcommon.InstrumentationLibrary
}

func newInstrumentationLibrary(orig **otlpcommon.InstrumentationLibrary) InstrumentationLibrary {
	return InstrumentationLibrary{orig}
}

// NewInstrumentationLibrary creates a new "nil" InstrumentationLibrary.
// To initialize the struct call "InitEmpty".
//
// This must be used only in testing code since no "Set" method available.
func NewInstrumentationLibrary() InstrumentationLibrary {
	orig := (*otlpcommon.InstrumentationLibrary)(nil)
	return newInstrumentationLibrary(&orig)
}

// InitEmpty overwrites the current value with empty.
func (ms InstrumentationLibrary) InitEmpty() {
	*ms.orig = &otlpcommon.InstrumentationLibrary{}
}

// IsNil returns true if the underlying data are nil.
// 
// Important: All other functions will cause a runtime error if this returns "true".
func (ms InstrumentationLibrary) IsNil() bool {
	return *ms.orig == nil
}

// Name returns the name associated with this InstrumentationLibrary.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms InstrumentationLibrary) Name() string {
	return (*ms.orig).Name
}

// SetName replaces the name associated with this InstrumentationLibrary.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms InstrumentationLibrary) SetName(v string) {
	(*ms.orig).Name = v
}

// Version returns the version associated with this InstrumentationLibrary.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms InstrumentationLibrary) Version() string {
	return (*ms.orig).Version
}

// SetVersion replaces the version associated with this InstrumentationLibrary.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms InstrumentationLibrary) SetVersion(v string) {
	(*ms.orig).Version = v
}
