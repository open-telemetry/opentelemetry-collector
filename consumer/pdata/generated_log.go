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

// Code generated by "cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "go run cmd/pdatagen/main.go".

package pdata

import (
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
)

// ResourceLogsSlice logically represents a slice of ResourceLogs.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewResourceLogsSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceLogsSlice struct {
	// orig points to the slice otlplogs.ResourceLogs field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like Resize.
	orig *[]*otlplogs.ResourceLogs
}

func newResourceLogsSlice(orig *[]*otlplogs.ResourceLogs) ResourceLogsSlice {
	return ResourceLogsSlice{orig}
}

// NewResourceLogsSlice creates a ResourceLogsSlice with 0 elements.
// Can use "Resize" to initialize with a given length.
func NewResourceLogsSlice() ResourceLogsSlice {
	orig := []*otlplogs.ResourceLogs(nil)
	return ResourceLogsSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewResourceLogsSlice()".
func (es ResourceLogsSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     ... // Do something with the element
// }
func (es ResourceLogsSlice) At(ix int) ResourceLogs {
	return newResourceLogs(&(*es.orig)[ix])
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es ResourceLogsSlice) MoveAndAppendTo(dest ResourceLogsSlice) {
	if es.Len() == 0 {
		// Just to ensure that we always return a Slice with nil elements.
		*es.orig = nil
		return
	}
	if dest.Len() == 0 {
		*dest.orig = *es.orig
		*es.orig = nil
		return
	}
	*dest.orig = append(*dest.orig, *es.orig...)
	*es.orig = nil
	return
}

// CopyTo copies all elements from the current slice to the dest.
func (es ResourceLogsSlice) CopyTo(dest ResourceLogsSlice) {
	newLen := es.Len()
	if newLen == 0 {
		*dest.orig = []*otlplogs.ResourceLogs(nil)
		return
	}
	oldLen := dest.Len()
	if newLen <= oldLen {
		(*dest.orig) = (*dest.orig)[:newLen]
		for i, el := range *es.orig {
			newResourceLogs(&el).CopyTo(newResourceLogs(&(*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlplogs.ResourceLogs, newLen)
	wrappers := make([]*otlplogs.ResourceLogs, newLen)
	for i, el := range *es.orig {
		wrappers[i] = &origs[i]
		newResourceLogs(&el).CopyTo(newResourceLogs(&wrappers[i]))
	}
	*dest.orig = wrappers
}

// Resize is an operation that resizes the slice:
// 1. If newLen is 0 then the slice is replaced with a nil slice.
// 2. If the newLen <= len then equivalent with slice[0:newLen].
// 3. If the newLen > len then (newLen - len) empty elements will be appended to the slice.
//
// Here is how a new ResourceLogsSlice can be initialized:
// es := NewResourceLogsSlice()
// es.Resize(4)
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     // Here should set all the values for e.
// }
func (es ResourceLogsSlice) Resize(newLen int) {
	if newLen == 0 {
		(*es.orig) = []*otlplogs.ResourceLogs(nil)
		return
	}
	oldLen := len(*es.orig)
	if newLen <= oldLen {
		(*es.orig) = (*es.orig)[:newLen]
		return
	}
	// TODO: Benchmark and optimize this logic.
	extraOrigs := make([]otlplogs.ResourceLogs, newLen-oldLen)
	oldOrig := (*es.orig)
	for i := range extraOrigs {
		oldOrig = append(oldOrig, &extraOrigs[i])
	}
	(*es.orig) = oldOrig
}

// Append will increase the length of the ResourceLogsSlice by one and set the
// given ResourceLogs at that new position.  The original ResourceLogs
// could still be referenced so do not reuse it after passing it to this
// method.
func (es ResourceLogsSlice) Append(e ResourceLogs) {
	*es.orig = append(*es.orig, *e.orig)
}

// ResourceLogs is a collection of logs from a Resource.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewResourceLogs function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceLogs struct {
	// orig points to the pointer otlplogs.ResourceLogs field contained somewhere else.
	// We use pointer-to-pointer to be able to modify it in InitEmpty func.
	orig **otlplogs.ResourceLogs
}

func newResourceLogs(orig **otlplogs.ResourceLogs) ResourceLogs {
	return ResourceLogs{orig}
}

// NewResourceLogs creates a new "nil" ResourceLogs.
// To initialize the struct call "InitEmpty".
//
// This must be used only in testing code since no "Set" method available.
func NewResourceLogs() ResourceLogs {
	orig := (*otlplogs.ResourceLogs)(nil)
	return newResourceLogs(&orig)
}

// InitEmpty overwrites the current value with empty.
func (ms ResourceLogs) InitEmpty() {
	*ms.orig = &otlplogs.ResourceLogs{}
}

// IsNil returns true if the underlying data are nil.
//
// Important: All other functions will cause a runtime error if this returns "true".
func (ms ResourceLogs) IsNil() bool {
	return *ms.orig == nil
}

// Resource returns the resource associated with this ResourceLogs.
// If no resource available, it creates an empty message and associates it with this ResourceLogs.
//
//  Empty initialized ResourceLogs will return "nil" Resource.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ResourceLogs) Resource() Resource {
	return newResource(&(*ms.orig).Resource)
}

// InstrumentationLibraryLogs returns the InstrumentationLibraryLogs associated with this ResourceLogs.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ResourceLogs) InstrumentationLibraryLogs() InstrumentationLibraryLogsSlice {
	return newInstrumentationLibraryLogsSlice(&(*ms.orig).InstrumentationLibraryLogs)
}

// CopyTo copies all properties from the current struct to the dest.
func (ms ResourceLogs) CopyTo(dest ResourceLogs) {
	if ms.IsNil() {
		*dest.orig = nil
		return
	}
	if dest.IsNil() {
		dest.InitEmpty()
	}
	ms.Resource().CopyTo(dest.Resource())
	ms.InstrumentationLibraryLogs().CopyTo(dest.InstrumentationLibraryLogs())
}

// InstrumentationLibraryLogsSlice logically represents a slice of InstrumentationLibraryLogs.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewInstrumentationLibraryLogsSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type InstrumentationLibraryLogsSlice struct {
	// orig points to the slice otlplogs.InstrumentationLibraryLogs field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like Resize.
	orig *[]*otlplogs.InstrumentationLibraryLogs
}

func newInstrumentationLibraryLogsSlice(orig *[]*otlplogs.InstrumentationLibraryLogs) InstrumentationLibraryLogsSlice {
	return InstrumentationLibraryLogsSlice{orig}
}

// NewInstrumentationLibraryLogsSlice creates a InstrumentationLibraryLogsSlice with 0 elements.
// Can use "Resize" to initialize with a given length.
func NewInstrumentationLibraryLogsSlice() InstrumentationLibraryLogsSlice {
	orig := []*otlplogs.InstrumentationLibraryLogs(nil)
	return InstrumentationLibraryLogsSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewInstrumentationLibraryLogsSlice()".
func (es InstrumentationLibraryLogsSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     ... // Do something with the element
// }
func (es InstrumentationLibraryLogsSlice) At(ix int) InstrumentationLibraryLogs {
	return newInstrumentationLibraryLogs(&(*es.orig)[ix])
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es InstrumentationLibraryLogsSlice) MoveAndAppendTo(dest InstrumentationLibraryLogsSlice) {
	if es.Len() == 0 {
		// Just to ensure that we always return a Slice with nil elements.
		*es.orig = nil
		return
	}
	if dest.Len() == 0 {
		*dest.orig = *es.orig
		*es.orig = nil
		return
	}
	*dest.orig = append(*dest.orig, *es.orig...)
	*es.orig = nil
	return
}

// CopyTo copies all elements from the current slice to the dest.
func (es InstrumentationLibraryLogsSlice) CopyTo(dest InstrumentationLibraryLogsSlice) {
	newLen := es.Len()
	if newLen == 0 {
		*dest.orig = []*otlplogs.InstrumentationLibraryLogs(nil)
		return
	}
	oldLen := dest.Len()
	if newLen <= oldLen {
		(*dest.orig) = (*dest.orig)[:newLen]
		for i, el := range *es.orig {
			newInstrumentationLibraryLogs(&el).CopyTo(newInstrumentationLibraryLogs(&(*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlplogs.InstrumentationLibraryLogs, newLen)
	wrappers := make([]*otlplogs.InstrumentationLibraryLogs, newLen)
	for i, el := range *es.orig {
		wrappers[i] = &origs[i]
		newInstrumentationLibraryLogs(&el).CopyTo(newInstrumentationLibraryLogs(&wrappers[i]))
	}
	*dest.orig = wrappers
}

// Resize is an operation that resizes the slice:
// 1. If newLen is 0 then the slice is replaced with a nil slice.
// 2. If the newLen <= len then equivalent with slice[0:newLen].
// 3. If the newLen > len then (newLen - len) empty elements will be appended to the slice.
//
// Here is how a new InstrumentationLibraryLogsSlice can be initialized:
// es := NewInstrumentationLibraryLogsSlice()
// es.Resize(4)
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     // Here should set all the values for e.
// }
func (es InstrumentationLibraryLogsSlice) Resize(newLen int) {
	if newLen == 0 {
		(*es.orig) = []*otlplogs.InstrumentationLibraryLogs(nil)
		return
	}
	oldLen := len(*es.orig)
	if newLen <= oldLen {
		(*es.orig) = (*es.orig)[:newLen]
		return
	}
	// TODO: Benchmark and optimize this logic.
	extraOrigs := make([]otlplogs.InstrumentationLibraryLogs, newLen-oldLen)
	oldOrig := (*es.orig)
	for i := range extraOrigs {
		oldOrig = append(oldOrig, &extraOrigs[i])
	}
	(*es.orig) = oldOrig
}

// Append will increase the length of the InstrumentationLibraryLogsSlice by one and set the
// given InstrumentationLibraryLogs at that new position.  The original InstrumentationLibraryLogs
// could still be referenced so do not reuse it after passing it to this
// method.
func (es InstrumentationLibraryLogsSlice) Append(e InstrumentationLibraryLogs) {
	*es.orig = append(*es.orig, *e.orig)
}

// InstrumentationLibraryLogs is a collection of logs from a LibraryInstrumentation.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewInstrumentationLibraryLogs function to create new instances.
// Important: zero-initialized instance is not valid for use.
type InstrumentationLibraryLogs struct {
	// orig points to the pointer otlplogs.InstrumentationLibraryLogs field contained somewhere else.
	// We use pointer-to-pointer to be able to modify it in InitEmpty func.
	orig **otlplogs.InstrumentationLibraryLogs
}

func newInstrumentationLibraryLogs(orig **otlplogs.InstrumentationLibraryLogs) InstrumentationLibraryLogs {
	return InstrumentationLibraryLogs{orig}
}

// NewInstrumentationLibraryLogs creates a new "nil" InstrumentationLibraryLogs.
// To initialize the struct call "InitEmpty".
//
// This must be used only in testing code since no "Set" method available.
func NewInstrumentationLibraryLogs() InstrumentationLibraryLogs {
	orig := (*otlplogs.InstrumentationLibraryLogs)(nil)
	return newInstrumentationLibraryLogs(&orig)
}

// InitEmpty overwrites the current value with empty.
func (ms InstrumentationLibraryLogs) InitEmpty() {
	*ms.orig = &otlplogs.InstrumentationLibraryLogs{}
}

// IsNil returns true if the underlying data are nil.
//
// Important: All other functions will cause a runtime error if this returns "true".
func (ms InstrumentationLibraryLogs) IsNil() bool {
	return *ms.orig == nil
}

// InstrumentationLibrary returns the instrumentationlibrary associated with this InstrumentationLibraryLogs.
// If no instrumentationlibrary available, it creates an empty message and associates it with this InstrumentationLibraryLogs.
//
//  Empty initialized InstrumentationLibraryLogs will return "nil" InstrumentationLibrary.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms InstrumentationLibraryLogs) InstrumentationLibrary() InstrumentationLibrary {
	return newInstrumentationLibrary(&(*ms.orig).InstrumentationLibrary)
}

// Logs returns the Logs associated with this InstrumentationLibraryLogs.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms InstrumentationLibraryLogs) Logs() LogSlice {
	return newLogSlice(&(*ms.orig).Logs)
}

// CopyTo copies all properties from the current struct to the dest.
func (ms InstrumentationLibraryLogs) CopyTo(dest InstrumentationLibraryLogs) {
	if ms.IsNil() {
		*dest.orig = nil
		return
	}
	if dest.IsNil() {
		dest.InitEmpty()
	}
	ms.InstrumentationLibrary().CopyTo(dest.InstrumentationLibrary())
	ms.Logs().CopyTo(dest.Logs())
}

// LogSlice logically represents a slice of LogRecord.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewLogSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type LogSlice struct {
	// orig points to the slice otlplogs.LogRecord field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like Resize.
	orig *[]*otlplogs.LogRecord
}

func newLogSlice(orig *[]*otlplogs.LogRecord) LogSlice {
	return LogSlice{orig}
}

// NewLogSlice creates a LogSlice with 0 elements.
// Can use "Resize" to initialize with a given length.
func NewLogSlice() LogSlice {
	orig := []*otlplogs.LogRecord(nil)
	return LogSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewLogSlice()".
func (es LogSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     ... // Do something with the element
// }
func (es LogSlice) At(ix int) LogRecord {
	return newLogRecord(&(*es.orig)[ix])
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es LogSlice) MoveAndAppendTo(dest LogSlice) {
	if es.Len() == 0 {
		// Just to ensure that we always return a Slice with nil elements.
		*es.orig = nil
		return
	}
	if dest.Len() == 0 {
		*dest.orig = *es.orig
		*es.orig = nil
		return
	}
	*dest.orig = append(*dest.orig, *es.orig...)
	*es.orig = nil
	return
}

// CopyTo copies all elements from the current slice to the dest.
func (es LogSlice) CopyTo(dest LogSlice) {
	newLen := es.Len()
	if newLen == 0 {
		*dest.orig = []*otlplogs.LogRecord(nil)
		return
	}
	oldLen := dest.Len()
	if newLen <= oldLen {
		(*dest.orig) = (*dest.orig)[:newLen]
		for i, el := range *es.orig {
			newLogRecord(&el).CopyTo(newLogRecord(&(*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlplogs.LogRecord, newLen)
	wrappers := make([]*otlplogs.LogRecord, newLen)
	for i, el := range *es.orig {
		wrappers[i] = &origs[i]
		newLogRecord(&el).CopyTo(newLogRecord(&wrappers[i]))
	}
	*dest.orig = wrappers
}

// Resize is an operation that resizes the slice:
// 1. If newLen is 0 then the slice is replaced with a nil slice.
// 2. If the newLen <= len then equivalent with slice[0:newLen].
// 3. If the newLen > len then (newLen - len) empty elements will be appended to the slice.
//
// Here is how a new LogSlice can be initialized:
// es := NewLogSlice()
// es.Resize(4)
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     // Here should set all the values for e.
// }
func (es LogSlice) Resize(newLen int) {
	if newLen == 0 {
		(*es.orig) = []*otlplogs.LogRecord(nil)
		return
	}
	oldLen := len(*es.orig)
	if newLen <= oldLen {
		(*es.orig) = (*es.orig)[:newLen]
		return
	}
	// TODO: Benchmark and optimize this logic.
	extraOrigs := make([]otlplogs.LogRecord, newLen-oldLen)
	oldOrig := (*es.orig)
	for i := range extraOrigs {
		oldOrig = append(oldOrig, &extraOrigs[i])
	}
	(*es.orig) = oldOrig
}

// Append will increase the length of the LogSlice by one and set the
// given LogRecord at that new position.  The original LogRecord
// could still be referenced so do not reuse it after passing it to this
// method.
func (es LogSlice) Append(e LogRecord) {
	*es.orig = append(*es.orig, *e.orig)
}

// LogRecord are experimental implementation of OpenTelemetry Log Data Model.

//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewLogRecord function to create new instances.
// Important: zero-initialized instance is not valid for use.
type LogRecord struct {
	// orig points to the pointer otlplogs.LogRecord field contained somewhere else.
	// We use pointer-to-pointer to be able to modify it in InitEmpty func.
	orig **otlplogs.LogRecord
}

func newLogRecord(orig **otlplogs.LogRecord) LogRecord {
	return LogRecord{orig}
}

// NewLogRecord creates a new "nil" LogRecord.
// To initialize the struct call "InitEmpty".
//
// This must be used only in testing code since no "Set" method available.
func NewLogRecord() LogRecord {
	orig := (*otlplogs.LogRecord)(nil)
	return newLogRecord(&orig)
}

// InitEmpty overwrites the current value with empty.
func (ms LogRecord) InitEmpty() {
	*ms.orig = &otlplogs.LogRecord{}
}

// IsNil returns true if the underlying data are nil.
//
// Important: All other functions will cause a runtime error if this returns "true".
func (ms LogRecord) IsNil() bool {
	return *ms.orig == nil
}

// Timestamp returns the timestamp associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) Timestamp() TimestampUnixNano {
	return TimestampUnixNano((*ms.orig).TimeUnixNano)
}

// SetTimestamp replaces the timestamp associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetTimestamp(v TimestampUnixNano) {
	(*ms.orig).TimeUnixNano = uint64(v)
}

// TraceID returns the traceid associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) TraceID() TraceID {
	return TraceID((*ms.orig).TraceId)
}

// SetTraceID replaces the traceid associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetTraceID(v TraceID) {
	(*ms.orig).TraceId = otlpcommon.TraceID(v)
}

// SpanID returns the spanid associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SpanID() SpanID {
	return SpanID((*ms.orig).SpanId)
}

// SetSpanID replaces the spanid associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetSpanID(v SpanID) {
	(*ms.orig).SpanId = []byte(v)
}

// Flags returns the flags associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) Flags() uint32 {
	return uint32((*ms.orig).Flags)
}

// SetFlags replaces the flags associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetFlags(v uint32) {
	(*ms.orig).Flags = uint32(v)
}

// SeverityText returns the severitytext associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SeverityText() string {
	return (*ms.orig).SeverityText
}

// SetSeverityText replaces the severitytext associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetSeverityText(v string) {
	(*ms.orig).SeverityText = v
}

// SeverityNumber returns the severitynumber associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SeverityNumber() SeverityNumber {
	return SeverityNumber((*ms.orig).SeverityNumber)
}

// SetSeverityNumber replaces the severitynumber associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetSeverityNumber(v SeverityNumber) {
	(*ms.orig).SeverityNumber = otlplogs.SeverityNumber(v)
}

// Name returns the name associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) Name() string {
	return (*ms.orig).Name
}

// SetName replaces the name associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetName(v string) {
	(*ms.orig).Name = v
}

// Body returns the body associated with this LogRecord.
// If no body available, it creates an empty message and associates it with this LogRecord.
//
//  Empty initialized LogRecord will return "nil" AttributeValue.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) Body() AttributeValue {
	return newAttributeValue(&(*ms.orig).Body)
}

// Attributes returns the Attributes associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) Attributes() AttributeMap {
	return newAttributeMap(&(*ms.orig).Attributes)
}

// DroppedAttributesCount returns the droppedattributescount associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) DroppedAttributesCount() uint32 {
	return (*ms.orig).DroppedAttributesCount
}

// SetDroppedAttributesCount replaces the droppedattributescount associated with this LogRecord.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms LogRecord) SetDroppedAttributesCount(v uint32) {
	(*ms.orig).DroppedAttributesCount = v
}

// CopyTo copies all properties from the current struct to the dest.
func (ms LogRecord) CopyTo(dest LogRecord) {
	if ms.IsNil() {
		*dest.orig = nil
		return
	}
	if dest.IsNil() {
		dest.InitEmpty()
	}
	dest.SetTimestamp(ms.Timestamp())
	dest.SetTraceID(ms.TraceID())
	dest.SetSpanID(ms.SpanID())
	dest.SetFlags(ms.Flags())
	dest.SetSeverityText(ms.SeverityText())
	dest.SetSeverityNumber(ms.SeverityNumber())
	dest.SetName(ms.Name())
	ms.Body().CopyTo(dest.Body())
	ms.Attributes().CopyTo(dest.Attributes())
	dest.SetDroppedAttributesCount(ms.DroppedAttributesCount())
}
