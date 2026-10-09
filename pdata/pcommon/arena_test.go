// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

// An arena is a []byte, so the garbage collector never scans it for pointers. Anything stored
// there must point at arena or wire memory, otherwise the bytes behind it can be freed while the
// slice still refers to them. Checking that the string was copied is deterministic, where
// observing the use-after-free would depend on GC timing.
func TestStringSliceAppendInternsIntoArena(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	var orig []string
	ss := StringSlice(internal.NewStringSliceWrapper(&orig, internal.NewState()))

	caller := strings.Clone("caller-owned-value")
	ss.Append(caller)
	require.Equal(t, 1, ss.Len())
	assert.Equal(t, caller, ss.At(0))
	assert.NotEqual(t, uintptr(unsafe.Pointer(unsafe.StringData(caller))), uintptr(unsafe.Pointer(unsafe.StringData(ss.At(0)))),
		"Append stored the caller's string instead of interning it into the arena")

	other := strings.Clone("another-caller-value")
	ss.SetAt(0, other)
	assert.Equal(t, other, ss.At(0))
	assert.NotEqual(t, uintptr(unsafe.Pointer(unsafe.StringData(other))), uintptr(unsafe.Pointer(unsafe.StringData(ss.At(0)))),
		"SetAt stored the caller's string instead of interning it into the arena")
}

func TestStringSliceAppendKeepsCallerStringWithoutArena(t *testing.T) {
	prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), false))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
	})

	var orig []string
	ss := StringSlice(internal.NewStringSliceWrapper(&orig, internal.NewState()))

	caller := strings.Clone("caller-owned-value")
	ss.Append(caller)
	require.Equal(t, 1, ss.Len())

	// Without an arena there is nothing to intern into and strings are immutable, so the
	// caller's string is stored as is, exactly as a plain append would.
	assert.Equal(t, uintptr(unsafe.Pointer(unsafe.StringData(caller))), uintptr(unsafe.Pointer(unsafe.StringData(ss.At(0)))))
}
