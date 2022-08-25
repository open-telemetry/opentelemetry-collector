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

package plog

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const traceFlagsMask = uint32(0xFF)

var DefaultLogRecordFlags = NewLogRecordFlagsFromRaw(uint32(0))

// LogRecordFlags defines flags for the LogRecord. 8 least significant bits are the trace flags as
// defined in W3C Trace Context specification. 24 most significant bits are reserved and must be set to 0.
//
// This is a reference type, if passed by value and callee modifies it the caller will see the modification.
//
// Must use NewLogRecordFlags function to create new instances.
// Important: zero-initialized instance is not valid for use.
type LogRecordFlags struct {
	orig uint32
}

// NewLogRecordFlagsFromRaw creates a new empty LogRecordFlags.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewLogRecordFlagsFromRaw(val uint32) LogRecordFlags {
	return LogRecordFlags{orig: val}
}

// TraceFlags copies all properties from the current struct to the dest.
func (ms LogRecordFlags) TraceFlags() pcommon.TraceFlags {
	return pcommon.NewTraceFlagsFromRaw(uint8(ms.AsRaw() & traceFlagsMask))
}

// SetTraceFlags moves all properties from the current struct to dest resetting the current instance to its zero value
func (ms LogRecordFlags) WithTraceFlags(v pcommon.TraceFlags) LogRecordFlags {
	orig := ms.orig
	orig &= ^traceFlagsMask   // cleanup old trace flags
	orig |= uint32(v.AsRaw()) // set new trace flags
	return LogRecordFlags{orig: orig}
}

// AsRaw converts LogRecordFlags to the OTLP uint32 representation.
func (ms LogRecordFlags) AsRaw() uint32 {
	return ms.orig
}
