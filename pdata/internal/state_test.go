// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssertMutable(t *testing.T) {
	assert.NotPanics(t, func() { NewState().AssertMutable() })
	assert.Panics(t, func() {
		state := NewState()
		state.MarkReadOnly()
		state.AssertMutable()
	})
}

func BenchmarkAssertMutable(b *testing.B) {
	b.ReportAllocs()

	mutable := NewState()
	for b.Loop() {
		mutable.AssertMutable()
	}
}
