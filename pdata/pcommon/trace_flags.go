package pcommon

const isSampledMask = uint8(1)

var DefaultTraceFlags = NewTraceFlagsFromRaw(0)

// TraceFlags defines the trace flags as defined by the w3c trace-context, see
// https://www.w3.org/TR/trace-context/#trace-flags
type TraceFlags struct {
	orig uint8
}

// NewTraceFlagsFromRaw creates a new empty TraceFlags.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewTraceFlagsFromRaw(val uint8) TraceFlags {
	return TraceFlags{orig: val}
}

// IsSampled returns true if the LogRecordFlags contains the IsSampled flag.
func (ms TraceFlags) IsSampled() bool {
	return ms.orig&isSampledMask != 0
}

// WithIsSampled returns a new TraceFlags, with the IsSampled flag set to the given value.
func (ms TraceFlags) WithIsSampled(b bool) TraceFlags {
	orig := ms.orig
	if b {
		orig |= isSampledMask
	} else {
		orig &^= isSampledMask
	}
	return NewTraceFlagsFromRaw(orig)
}

// AsRaw converts LogRecordFlags to the OTLP uint32 representation.
func (ms TraceFlags) AsRaw() uint8 {
	return ms.orig
}
