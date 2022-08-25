// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pcommon // import "go.opentelemetry.io/collector/pdata/pcommon"

const isSampledMask = uint8(1)

var DefaultTraceFlags = TraceFlags(0)

// TraceFlags defines the trace flags as defined by the w3c trace-context, see
// https://www.w3.org/TR/trace-context/#trace-flags
type TraceFlags uint8

// IsSampled returns true if the TraceFlags contains the IsSampled flag.
func (ms TraceFlags) IsSampled() bool {
	return uint8(ms)&isSampledMask != 0
}

// WithIsSampled returns a new TraceFlags, with the IsSampled flag set to the given value.
func (ms TraceFlags) WithIsSampled(b bool) TraceFlags {
	orig := uint8(ms)
	if b {
		orig |= isSampledMask
	} else {
		orig &^= isSampledMask
	}
	return TraceFlags(orig)
}
