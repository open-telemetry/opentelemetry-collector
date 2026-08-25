// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestTraceState_MoveTo(t *testing.T) {
	ms := TraceState(internal.GenTestTraceStateWrapper())
	dest := NewTraceState()
	ms.MoveTo(dest)
	assert.Equal(t, NewTraceState(), ms)
	assert.Equal(t, TraceState(internal.GenTestTraceStateWrapper()), dest)

	dest.MoveTo(dest)
	assert.Equal(t, TraceState(internal.GenTestTraceStateWrapper()), dest)
}

func TestTraceState_CopyTo(t *testing.T) {
	ms := NewTraceState()
	orig := NewTraceState()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = TraceState(internal.GenTestTraceStateWrapper())
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}

func TestTraceState_FromRaw_AsRaw(t *testing.T) {
	ms := NewTraceState()
	assert.Empty(t, ms.AsRaw())
	ms.FromRaw("congo=t61rcWkgMzE")
	assert.Equal(t, "congo=t61rcWkgMzE", ms.AsRaw())
}

func TestInvalidTraceState(t *testing.T) {
	v := TraceState{}

	assert.Panics(t, func() { v.AsRaw() })
	assert.Panics(t, func() { v.FromRaw("") })
	assert.Panics(t, func() { v.MoveTo(TraceState{}) })
	assert.Panics(t, func() { v.CopyTo(TraceState{}) })
}
