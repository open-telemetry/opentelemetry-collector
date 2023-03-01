// Copyright The OpenTelemetry Authors
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

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pmetric

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ResourceMetrics is a collection of metrics from a Resource.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewResourceMetrics function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceMetrics struct {
	*pResourceMetrics
}

type pResourceMetrics struct {
	orig   *otlpmetrics.ResourceMetrics
	state  *internal.State
	parent ResourceMetricsSlice
	idx    int
}

func (ms ResourceMetrics) getOrig() *otlpmetrics.ResourceMetrics {
	if *ms.state == internal.StateDirty {
		ms.orig, ms.state = ms.parent.refreshElementOrigState(ms.idx)
	}
	return ms.orig
}

func (ms ResourceMetrics) ensureMutability() {
	if *ms.state == internal.StateShared {
		ms.parent.ensureMutability()
	}
}

func (ms ResourceMetrics) getState() *internal.State {
	return ms.state
}

type wrappedResourceMetricsResource struct {
	ResourceMetrics
}

func (es wrappedResourceMetricsResource) RefreshOrigState() (*otlpresource.Resource, *internal.State) {
	return &es.getOrig().Resource, es.getState()
}

func (es wrappedResourceMetricsResource) EnsureMutability() {
	es.ensureMutability()
}

func (es wrappedResourceMetricsResource) GetState() *internal.State {
	return es.getState()
}

func (ms ResourceMetrics) refreshScopeMetricsOrigState() (*[]*otlpmetrics.ScopeMetrics, *internal.State) {
	return &ms.getOrig().ScopeMetrics, ms.state
}

func newResourceMetrics(orig *otlpmetrics.ResourceMetrics, parent ResourceMetricsSlice, idx int) ResourceMetrics {
	return ResourceMetrics{&pResourceMetrics{
		orig:   orig,
		state:  parent.getState(),
		parent: parent,
		idx:    idx,
	}}
}

// NewResourceMetrics creates a new empty ResourceMetrics.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewResourceMetrics() ResourceMetrics {
	state := internal.StateExclusive
	return ResourceMetrics{&pResourceMetrics{orig: &otlpmetrics.ResourceMetrics{}, state: &state}}
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms ResourceMetrics) MoveTo(dest ResourceMetrics) {
	ms.ensureMutability()
	dest.ensureMutability()
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpmetrics.ResourceMetrics{}
}

// Resource returns the resource associated with this ResourceMetrics.
func (ms ResourceMetrics) Resource() pcommon.Resource {
	return pcommon.Resource(internal.NewResource(&ms.getOrig().Resource, wrappedResourceMetricsResource{ResourceMetrics: ms}))
}

// SchemaUrl returns the schemaurl associated with this ResourceMetrics.
func (ms ResourceMetrics) SchemaUrl() string {
	return ms.getOrig().SchemaUrl
}

// SetSchemaUrl replaces the schemaurl associated with this ResourceMetrics.
func (ms ResourceMetrics) SetSchemaUrl(v string) {
	ms.ensureMutability()
	ms.getOrig().SchemaUrl = v
}

// ScopeMetrics returns the <no value> associated with this ResourceMetrics.
func (ms ResourceMetrics) ScopeMetrics() ScopeMetricsSlice {
	return newScopeMetricsSlice(&ms.getOrig().ScopeMetrics, ms)
}

// CopyTo copies all properties from the current struct overriding the destination.
func (ms ResourceMetrics) CopyTo(dest ResourceMetrics) {
	dest.ensureMutability()
	ms.Resource().CopyTo(dest.Resource())
	dest.SetSchemaUrl(ms.SchemaUrl())
	ms.ScopeMetrics().CopyTo(dest.ScopeMetrics())
}
