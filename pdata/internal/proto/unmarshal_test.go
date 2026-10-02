// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import "testing"

func TestCountField(t *testing.T) {
	// field 1, wire type varint, value 1, three times.
	buf := []byte{0x08, 0x01, 0x08, 0x01, 0x08, 0x01}
	if got := CountField(buf, 0, 1); got != 3 {
		t.Fatalf("CountField from start = %d, want 3", got)
	}
	if got := CountField(buf, 2, 1); got != 2 {
		t.Fatalf("CountField from second = %d, want 2", got)
	}
	if got := CountField(buf, len(buf), 1); got != 0 {
		t.Fatalf("CountField at end = %d, want 0", got)
	}
}
