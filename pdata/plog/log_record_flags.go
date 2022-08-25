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

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const traceFlagsMask = uint32(0xFF)

var DefaultLogRecordFlags = LogRecordFlags(0)

// LogRecordFlags defines flags for the LogRecord. 8 least significant bits are the trace flags as
// defined in W3C Trace Context specification. 24 most significant bits are reserved and must be set to 0.
type LogRecordFlags uint32

// TraceFlags returns the TraceFlags part of the LogRecordFlags.
func (ms LogRecordFlags) TraceFlags() pcommon.TraceFlags {
	return pcommon.TraceFlags(uint32(ms) & traceFlagsMask)
}

// WithTraceFlags returns a new LogRecordFlags, with the TraceFlags part set to the given value.
func (ms LogRecordFlags) WithTraceFlags(v pcommon.TraceFlags) LogRecordFlags {
	orig := uint32(ms)
	orig &= ^traceFlagsMask // cleanup old trace flags
	orig |= uint32(v)       // set new trace flags
	return LogRecordFlags(orig)
}
