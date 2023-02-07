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

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestSlice(t *testing.T) {
	es := NewMutableSlice()
	assert.Equal(t, 0, es.Len())
	es = NewMutableSliceFromOrig(&[]otlpcommon.AnyValue{})
	assert.Equal(t, 0, es.Len())

	es.EnsureCapacity(7)
	emptyVal := NewMutableValueFromOrig(&otlpcommon.AnyValue{})
	testVal := GenerateTestValue()
	assert.Equal(t, 7, cap(*es.orig))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, el)
		FillTestValue(el)
		assert.Equal(t, testVal, el)
	}
}

func TestSlice_CopyTo(t *testing.T) {
	dest := NewMutableSlice()
	// Test CopyTo to empty
	NewMutableSlice().CopyTo(dest)
	assert.Equal(t, NewMutableSlice(), dest)

	// Test CopyTo larger slice
	GenerateTestSlice().CopyTo(dest)
	assert.Equal(t, GenerateTestSlice(), dest)

	// Test CopyTo same size slice
	GenerateTestSlice().CopyTo(dest)
	assert.Equal(t, GenerateTestSlice(), dest)
}

func TestSlice_EnsureCapacity(t *testing.T) {
	es := GenerateTestSlice()
	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	expectedEs := make(map[*otlpcommon.AnyValue]bool)
	for i := 0; i < es.Len(); i++ {
		expectedEs[es.At(i).orig] = true
	}
	assert.Equal(t, es.Len(), len(expectedEs))
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	foundEs := make(map[*otlpcommon.AnyValue]bool, es.Len())
	for i := 0; i < es.Len(); i++ {
		foundEs[es.At(i).orig] = true
	}
	assert.Equal(t, expectedEs, foundEs)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	oldLen := es.Len()
	assert.Equal(t, oldLen, len(expectedEs))
	es.EnsureCapacity(ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
}

func TestSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := GenerateTestSlice()
	dest := NewMutableSlice()
	src := GenerateTestSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, GenerateTestSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, GenerateTestSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	GenerateTestSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewMutableSlice()
	emptySlice.RemoveIf(func(el MutableValue) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := GenerateTestSlice()
	pos := 0
	filtered.RemoveIf(func(el MutableValue) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}
