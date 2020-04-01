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
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
)

// Resource information.
//
// This is a reference type, if passsed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewResource function to create new instances.
// Important: zero-initialized instance is not valid for use.
type Resource struct {
	// Wrap OTLP otlpresource.Resource.
	orig **otlpresource.Resource
}

func newResource(orig **otlpresource.Resource) Resource {
	return Resource{orig}
}

// NewResource creates a new "nil" Resource.
// To initialize the struct call "InitEmpty".
//
// This must be used only in testing code since no "Set" method available.
func NewResource() Resource {
	orig := (*otlpresource.Resource)(nil)
	return newResource(&orig)
}

// InitEmpty overwrites the current value with empty.
func (ms Resource) InitEmpty() {
	*ms.orig = &otlpresource.Resource{}
}

// IsNil returns true if the underlying data are nil.
// 
// Important: All other functions will cause a runtime error if this returns "true".
func (ms Resource) IsNil() bool {
	return *ms.orig == nil
}

// Attributes returns the Attributes associated with this Resource.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms Resource) Attributes() AttributeMap {
	return newAttributeMap(&(*ms.orig).Attributes)
}

// SetAttributes replaces the Attributes associated with this Resource.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms Resource) SetAttributes(v AttributeMap) {
	(*ms.orig).Attributes = *v.orig
}
