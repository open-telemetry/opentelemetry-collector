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

package data

import (
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
)

// Resource information.
//
// Must use NewResource functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Resource struct {
	orig *otlpresource.Resource
}

// NewResource creates a new empty Resource.
func NewResource() Resource {
	return Resource{&otlpresource.Resource{}}
}

func newResource(orig *otlpresource.Resource) Resource {
	return Resource{orig}
}

// Attributes returns the AttributesMap associated with this Resource.
func (r Resource) Attributes() AttributeMap {
	return AttributeMap{&r.orig.Attributes}
}

// SetAttributes repaces the AttributesMap associated with this Resource.
func (r Resource) SetAttributes(v AttributeMap) {
	r.orig.Attributes = *v.orig
}
