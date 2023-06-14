// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

const isSampledMask = uint32(1)

var DefaultProfileRecordFlags = ProfileRecordFlags(0)

// ProfileRecordFlags defines flags for the ProfileRecord. The 8 least significant bits are the trace flags as
// defined in W3C Trace Context specification. 24 most significant bits are reserved and must be set to 0.
type ProfileRecordFlags uint32

// IsSampled returns true if the ProfileRecordFlags contains the IsSampled flag.
func (ms ProfileRecordFlags) IsSampled() bool {
	return uint32(ms)&isSampledMask != 0
}

// WithIsSampled returns a new ProfileRecordFlags, with the IsSampled flag set to the given value.
func (ms ProfileRecordFlags) WithIsSampled(b bool) ProfileRecordFlags {
	orig := uint32(ms)
	if b {
		orig |= isSampledMask
	} else {
		orig &^= isSampledMask
	}
	return ProfileRecordFlags(orig)
}
