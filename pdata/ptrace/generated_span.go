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

package ptrace

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/data"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Span represents a single operation within a trace.
// See Span definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto
type Span interface {
	commonSpan
	TraceState() pcommon.TraceState
	Attributes() pcommon.Map
	Events() SpanEventSlice
	Links() SpanLinkSlice
	Status() Status
}

type MutableSpan interface {
	commonSpan
	MoveTo(dest MutableSpan)
	SetTraceID(pcommon.TraceID)
	SetSpanID(pcommon.SpanID)
	TraceState() pcommon.MutableTraceState
	SetParentSpanID(pcommon.SpanID)
	SetName(string)
	SetKind(SpanKind)
	SetStartTimestamp(pcommon.Timestamp)
	SetEndTimestamp(pcommon.Timestamp)
	Attributes() pcommon.MutableMap
	SetDroppedAttributesCount(uint32)
	Events() MutableSpanEventSlice
	SetDroppedEventsCount(uint32)
	Links() MutableSpanLinkSlice
	SetDroppedLinksCount(uint32)
	Status() MutableStatus
}

type commonSpan interface {
	getOrig() *otlptrace.Span
	CopyTo(dest MutableSpan)
	TraceID() pcommon.TraceID
	SpanID() pcommon.SpanID
	ParentSpanID() pcommon.SpanID
	Name() string
	Kind() SpanKind
	StartTimestamp() pcommon.Timestamp
	EndTimestamp() pcommon.Timestamp
	DroppedAttributesCount() uint32
	DroppedEventsCount() uint32
	DroppedLinksCount() uint32
}

type immutableSpan struct {
	orig *otlptrace.Span
}

type mutableSpan struct {
	immutableSpan
}

func newImmutableSpan(orig *otlptrace.Span) immutableSpan {
	return immutableSpan{orig}
}

func newMutableSpan(orig *otlptrace.Span) mutableSpan {
	return mutableSpan{immutableSpan{orig}}
}

func (ms immutableSpan) getOrig() *otlptrace.Span {
	return ms.orig
}

// NewSpan creates a new empty Span.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewSpan() MutableSpan {
	return newMutableSpan(&otlptrace.Span{})
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms mutableSpan) MoveTo(dest MutableSpan) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlptrace.Span{}
}

// TraceID returns the traceid associated with this Span.
func (ms immutableSpan) TraceID() pcommon.TraceID {
	return pcommon.TraceID(ms.orig.TraceId)
}

// SetTraceID replaces the traceid associated with this Span.
func (ms mutableSpan) SetTraceID(v pcommon.TraceID) {
	ms.orig.TraceId = data.TraceID(v)
}

// SpanID returns the spanid associated with this Span.
func (ms immutableSpan) SpanID() pcommon.SpanID {
	return pcommon.SpanID(ms.orig.SpanId)
}

// SetSpanID replaces the spanid associated with this Span.
func (ms mutableSpan) SetSpanID(v pcommon.SpanID) {
	ms.orig.SpanId = data.SpanID(v)
}

// TraceState returns the tracestate associated with this Span.
func (ms immutableSpan) TraceState() pcommon.TraceState {
	return internal.NewImmutableTraceState(&ms.getOrig().TraceState)
}

// TraceState returns the tracestate associated with this Span.
func (ms mutableSpan) TraceState() pcommon.MutableTraceState {
	return internal.NewMutableTraceState(&ms.getOrig().TraceState)
}

// ParentSpanID returns the parentspanid associated with this Span.
func (ms immutableSpan) ParentSpanID() pcommon.SpanID {
	return pcommon.SpanID(ms.orig.ParentSpanId)
}

// SetParentSpanID replaces the parentspanid associated with this Span.
func (ms mutableSpan) SetParentSpanID(v pcommon.SpanID) {
	ms.orig.ParentSpanId = data.SpanID(v)
}

// Name returns the name associated with this Span.
func (ms immutableSpan) Name() string {
	return ms.getOrig().Name
}

// SetName replaces the name associated with this Span.
func (ms mutableSpan) SetName(v string) {
	ms.getOrig().Name = v
}

// Kind returns the kind associated with this Span.
func (ms immutableSpan) Kind() SpanKind {
	return SpanKind(ms.orig.Kind)
}

// SetKind replaces the kind associated with this Span.
func (ms mutableSpan) SetKind(v SpanKind) {
	ms.orig.Kind = otlptrace.Span_SpanKind(v)
}

// StartTimestamp returns the starttimestamp associated with this Span.
func (ms immutableSpan) StartTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(ms.orig.StartTimeUnixNano)
}

// SetStartTimestamp replaces the starttimestamp associated with this Span.
func (ms mutableSpan) SetStartTimestamp(v pcommon.Timestamp) {
	ms.orig.StartTimeUnixNano = uint64(v)
}

// EndTimestamp returns the endtimestamp associated with this Span.
func (ms immutableSpan) EndTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(ms.orig.EndTimeUnixNano)
}

// SetEndTimestamp replaces the endtimestamp associated with this Span.
func (ms mutableSpan) SetEndTimestamp(v pcommon.Timestamp) {
	ms.orig.EndTimeUnixNano = uint64(v)
}

// Attributes returns the Attributes associated with this Span.
func (ms immutableSpan) Attributes() pcommon.Map {
	return internal.NewImmutableMap(&ms.getOrig().Attributes)
}

func (ms mutableSpan) Attributes() pcommon.MutableMap {
	return internal.NewMutableMap(&ms.getOrig().Attributes)
}

// DroppedAttributesCount returns the droppedattributescount associated with this Span.
func (ms immutableSpan) DroppedAttributesCount() uint32 {
	return ms.getOrig().DroppedAttributesCount
}

// SetDroppedAttributesCount replaces the droppedattributescount associated with this Span.
func (ms mutableSpan) SetDroppedAttributesCount(v uint32) {
	ms.getOrig().DroppedAttributesCount = v
}

// Events returns the Events associated with this Span.
func (ms immutableSpan) Events() SpanEventSlice {
	return newImmutableSpanEventSlice(&ms.getOrig().Events)
}

func (ms mutableSpan) Events() MutableSpanEventSlice {
	return newMutableSpanEventSlice(&ms.getOrig().Events)
}

// DroppedEventsCount returns the droppedeventscount associated with this Span.
func (ms immutableSpan) DroppedEventsCount() uint32 {
	return ms.getOrig().DroppedEventsCount
}

// SetDroppedEventsCount replaces the droppedeventscount associated with this Span.
func (ms mutableSpan) SetDroppedEventsCount(v uint32) {
	ms.getOrig().DroppedEventsCount = v
}

// Links returns the Links associated with this Span.
func (ms immutableSpan) Links() SpanLinkSlice {
	return newImmutableSpanLinkSlice(&ms.getOrig().Links)
}

func (ms mutableSpan) Links() MutableSpanLinkSlice {
	return newMutableSpanLinkSlice(&ms.getOrig().Links)
}

// DroppedLinksCount returns the droppedlinkscount associated with this Span.
func (ms immutableSpan) DroppedLinksCount() uint32 {
	return ms.getOrig().DroppedLinksCount
}

// SetDroppedLinksCount replaces the droppedlinkscount associated with this Span.
func (ms mutableSpan) SetDroppedLinksCount(v uint32) {
	ms.getOrig().DroppedLinksCount = v
}

// Status returns the status associated with this Span.
func (ms immutableSpan) Status() Status {
	return newImmutableStatus(&ms.getOrig().Status)
}

// Status returns the status associated with this MutableSpan.
func (ms mutableSpan) Status() MutableStatus {
	return newMutableStatus(&ms.getOrig().Status)
}

// CopyTo copies all properties from the current struct overriding the destination.
func (ms immutableSpan) CopyTo(dest MutableSpan) {
	dest.SetTraceID(ms.TraceID())
	dest.SetSpanID(ms.SpanID())
	ms.TraceState().CopyTo(dest.TraceState())
	dest.SetParentSpanID(ms.ParentSpanID())
	dest.SetName(ms.Name())
	dest.SetKind(ms.Kind())
	dest.SetStartTimestamp(ms.StartTimestamp())
	dest.SetEndTimestamp(ms.EndTimestamp())
	ms.Attributes().CopyTo(dest.Attributes())
	dest.SetDroppedAttributesCount(ms.DroppedAttributesCount())
	ms.Events().CopyTo(dest.Events())
	dest.SetDroppedEventsCount(ms.DroppedEventsCount())
	ms.Links().CopyTo(dest.Links())
	dest.SetDroppedLinksCount(ms.DroppedLinksCount())
	ms.Status().CopyTo(dest.Status())
}

func generateTestSpan() MutableSpan {
	tv := NewSpan()
	fillTestSpan(tv)
	return tv
}

func fillTestSpan(tv MutableSpan) {
	tv.getOrig().TraceId = data.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	tv.getOrig().SpanId = data.SpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1})
	internal.FillTestTraceState(internal.NewTraceState(&tv.orig.TraceState))
	tv.getOrig().ParentSpanId = data.SpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1})
	tv.getOrig().Name = "test_name"
	tv.getOrig().Kind = otlptrace.Span_SpanKind(3)
	tv.getOrig().StartTimeUnixNano = 1234567890
	tv.getOrig().EndTimeUnixNano = 1234567890
	internal.FillTestMap(internal.NewMutableMap(&tv.getOrig().Attributes))
	tv.getOrig().DroppedAttributesCount = uint32(17)
	fillTestSpanEventSlice(newMutableSpanEventSlice(&tv.getOrig().Events))
	tv.getOrig().DroppedEventsCount = uint32(17)
	fillTestSpanLinkSlice(newMutableSpanLinkSlice(&tv.getOrig().Links))
	tv.getOrig().DroppedLinksCount = uint32(17)
	fillTestStatus(newStatus(&tv.orig.Status))
}
