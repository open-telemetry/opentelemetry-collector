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

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestSlice(t *testing.T) {
	es := NewSlice()
	assert.Equal(t, 0, es.Len())
	es = newSlice(&[]otlpcommon.AnyValue{})
	assert.Equal(t, 0, es.Len())

	es.EnsureCapacity(7)
	emptyVal := newValue(&otlpcommon.AnyValue{})
	testVal := Value(internal.GenerateTestValue())
	assert.Equal(t, 7, cap(*es.getOrig()))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, el)
		internal.FillTestValue(internal.Value(el))
		assert.Equal(t, testVal, el)
	}
}

func TestSlice_CopyTo(t *testing.T) {
	dest := NewSlice()
	// Test CopyTo to empty
	NewSlice().CopyTo(dest)
	assert.Equal(t, NewSlice(), dest)

	// Test CopyTo larger slice
	Slice(internal.GenerateTestSlice()).CopyTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)

	// Test CopyTo same size slice
	Slice(internal.GenerateTestSlice()).CopyTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)
}

func TestSlice_EnsureCapacity(t *testing.T) {
	es := Slice(internal.GenerateTestSlice())
	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	expectedEs := make(map[*otlpcommon.AnyValue]bool)
	for i := 0; i < es.Len(); i++ {
		expectedEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, es.Len(), len(expectedEs))
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	foundEs := make(map[*otlpcommon.AnyValue]bool, es.Len())
	for i := 0; i < es.Len(); i++ {
		foundEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, expectedEs, foundEs)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	oldLen := es.Len()
	assert.Equal(t, oldLen, len(expectedEs))
	es.EnsureCapacity(ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.getOrig()))
}

func TestSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := Slice(internal.GenerateTestSlice())
	dest := NewSlice()
	src := Slice(internal.GenerateTestSlice())
	src.MoveAndAppendTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	Slice(internal.GenerateTestSlice()).MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewSlice()
	emptySlice.RemoveIf(func(el Value) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := Slice(internal.GenerateTestSlice())
	pos := 0
	filtered.RemoveIf(func(el Value) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}
