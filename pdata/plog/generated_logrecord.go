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

package plog

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/data"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// LogRecord are experimental implementation of OpenTelemetry Log Data Model.

// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewLogRecord function to create new instances.
// Important: zero-initialized instance is not valid for use.
type LogRecord struct {
	*pLogRecord
}

type pLogRecord struct {
	orig   *otlplogs.LogRecord
	state  *internal.State
	parent LogRecordSlice
	idx    int
}

func (ms LogRecord) getOrig() *otlplogs.LogRecord {
	if *ms.state == internal.StateDirty {
		ms.orig, ms.state = ms.parent.refreshElementOrigState(ms.idx)
	}
	return ms.orig
}

func (ms LogRecord) ensureMutability() {
	if *ms.state == internal.StateShared {
		ms.parent.ensureMutability()
	}
}

func (ms LogRecord) getState() *internal.State {
	return ms.state
}

type wrappedLogRecordBody struct {
	LogRecord
}

func (es wrappedLogRecordBody) RefreshOrigState() (*otlpcommon.AnyValue, *internal.State) {
	return &es.getOrig().Body, es.getState()
}

func (es wrappedLogRecordBody) EnsureMutability() {
	es.ensureMutability()
}

func (es wrappedLogRecordBody) GetState() *internal.State {
	return es.getState()
}

type wrappedLogRecordAttributes struct {
	LogRecord
}

func (es wrappedLogRecordAttributes) RefreshOrigState() (*[]otlpcommon.KeyValue, *internal.State) {
	return &es.getOrig().Attributes, es.getState()
}

func (es wrappedLogRecordAttributes) EnsureMutability() {
	es.ensureMutability()
}

func (es wrappedLogRecordAttributes) GetState() *internal.State {
	return es.getState()
}

func newLogRecord(orig *otlplogs.LogRecord, parent LogRecordSlice, idx int) LogRecord {
	return LogRecord{&pLogRecord{
		orig:   orig,
		state:  parent.getState(),
		parent: parent,
		idx:    idx,
	}}
}

// NewLogRecord creates a new empty LogRecord.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewLogRecord() LogRecord {
	state := internal.StateExclusive
	return LogRecord{&pLogRecord{orig: &otlplogs.LogRecord{}, state: &state}}
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms LogRecord) MoveTo(dest LogRecord) {
	ms.ensureMutability()
	dest.ensureMutability()
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlplogs.LogRecord{}
}

// ObservedTimestamp returns the observedtimestamp associated with this LogRecord.
func (ms LogRecord) ObservedTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(ms.getOrig().ObservedTimeUnixNano)
}

// SetObservedTimestamp replaces the observedtimestamp associated with this LogRecord.
func (ms LogRecord) SetObservedTimestamp(v pcommon.Timestamp) {
	ms.ensureMutability()
	ms.getOrig().ObservedTimeUnixNano = uint64(v)
}

// Timestamp returns the timestamp associated with this LogRecord.
func (ms LogRecord) Timestamp() pcommon.Timestamp {
	return pcommon.Timestamp(ms.getOrig().TimeUnixNano)
}

// SetTimestamp replaces the timestamp associated with this LogRecord.
func (ms LogRecord) SetTimestamp(v pcommon.Timestamp) {
	ms.ensureMutability()
	ms.getOrig().TimeUnixNano = uint64(v)
}

// TraceID returns the traceid associated with this LogRecord.
func (ms LogRecord) TraceID() pcommon.TraceID {
	return pcommon.TraceID(ms.getOrig().TraceId)
}

// SetTraceID replaces the traceid associated with this LogRecord.
func (ms LogRecord) SetTraceID(v pcommon.TraceID) {
	ms.ensureMutability()
	ms.getOrig().TraceId = data.TraceID(v)
}

// SpanID returns the spanid associated with this LogRecord.
func (ms LogRecord) SpanID() pcommon.SpanID {
	return pcommon.SpanID(ms.getOrig().SpanId)
}

// SetSpanID replaces the spanid associated with this LogRecord.
func (ms LogRecord) SetSpanID(v pcommon.SpanID) {
	ms.ensureMutability()
	ms.getOrig().SpanId = data.SpanID(v)
}

// Flags returns the flags associated with this LogRecord.
func (ms LogRecord) Flags() LogRecordFlags {
	return LogRecordFlags(ms.getOrig().Flags)
}

// SetFlags replaces the flags associated with this LogRecord.
func (ms LogRecord) SetFlags(v LogRecordFlags) {
	ms.ensureMutability()
	ms.getOrig().Flags = uint32(v)
}

// SeverityText returns the severitytext associated with this LogRecord.
func (ms LogRecord) SeverityText() string {
	return ms.getOrig().SeverityText
}

// SetSeverityText replaces the severitytext associated with this LogRecord.
func (ms LogRecord) SetSeverityText(v string) {
	ms.ensureMutability()
	ms.getOrig().SeverityText = v
}

// SeverityNumber returns the severitynumber associated with this LogRecord.
func (ms LogRecord) SeverityNumber() SeverityNumber {
	return SeverityNumber(ms.getOrig().SeverityNumber)
}

// SetSeverityNumber replaces the severitynumber associated with this LogRecord.
func (ms LogRecord) SetSeverityNumber(v SeverityNumber) {
	ms.ensureMutability()
	ms.getOrig().SeverityNumber = otlplogs.SeverityNumber(v)
}

// Body returns the body associated with this LogRecord.
func (ms LogRecord) Body() pcommon.Value {
	return pcommon.Value(internal.NewValue(&ms.getOrig().Body, wrappedLogRecordBody{LogRecord: ms}))
}

// Attributes returns the <no value> associated with this LogRecord.
func (ms LogRecord) Attributes() pcommon.Map {
	return pcommon.Map(internal.NewMap(&ms.getOrig().Attributes, wrappedLogRecordAttributes{LogRecord: ms}))
}

// DroppedAttributesCount returns the droppedattributescount associated with this LogRecord.
func (ms LogRecord) DroppedAttributesCount() uint32 {
	return ms.getOrig().DroppedAttributesCount
}

// SetDroppedAttributesCount replaces the droppedattributescount associated with this LogRecord.
func (ms LogRecord) SetDroppedAttributesCount(v uint32) {
	ms.ensureMutability()
	ms.getOrig().DroppedAttributesCount = v
}

// CopyTo copies all properties from the current struct overriding the destination.
func (ms LogRecord) CopyTo(dest LogRecord) {
	dest.ensureMutability()
	dest.SetObservedTimestamp(ms.ObservedTimestamp())
	dest.SetTimestamp(ms.Timestamp())
	dest.SetTraceID(ms.TraceID())
	dest.SetSpanID(ms.SpanID())
	dest.SetFlags(ms.Flags())
	dest.SetSeverityText(ms.SeverityText())
	dest.SetSeverityNumber(ms.SeverityNumber())
	ms.Body().CopyTo(dest.Body())
	ms.Attributes().CopyTo(dest.Attributes())
	dest.SetDroppedAttributesCount(ms.DroppedAttributesCount())
}
