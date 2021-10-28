// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by "model/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "go run model/internal/cmd/pdatagen/main.go".

package pdata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlpcommon "go.opentelemetry.io/collector/model/internal/data/protogen/common/v1"
)

func TestInstrumentationLibrary_MoveTo(t *testing.T) {
	ms := generateTestInstrumentationLibrary()
	dest := NewInstrumentationLibrary()
	ms.MoveTo(dest)
	assert.EqualValues(t, NewInstrumentationLibrary(), ms)
	assert.EqualValues(t, generateTestInstrumentationLibrary(), dest)
}

func TestInstrumentationLibrary_CopyTo(t *testing.T) {
	ms := NewInstrumentationLibrary()
	generateTestInstrumentationLibrary().CopyTo(ms)
	assert.EqualValues(t, generateTestInstrumentationLibrary(), ms)
}

func TestInstrumentationLibrary_Name(t *testing.T) {
	ms := NewInstrumentationLibrary()
	assert.EqualValues(t, "", ms.Name())
	testValName := "test_name"
	ms.SetName(testValName)
	assert.EqualValues(t, testValName, ms.Name())
}

func TestInstrumentationLibrary_Version(t *testing.T) {
	ms := NewInstrumentationLibrary()
	assert.EqualValues(t, "", ms.Version())
	testValVersion := "test_version"
	ms.SetVersion(testValVersion)
	assert.EqualValues(t, testValVersion, ms.Version())
}

func TestAnyValueArray(t *testing.T) {
	es := NewAnyValueArray()
	assert.EqualValues(t, 0, es.Len())
	es = newAnyValueArray(&[]otlpcommon.AnyValue{})
	assert.EqualValues(t, 0, es.Len())

	es.EnsureCapacity(7)
	emptyVal := newAttributeValue(&otlpcommon.AnyValue{})
	testVal := generateTestAttributeValue()
	assert.EqualValues(t, 7, cap(*es.orig))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.EqualValues(t, emptyVal, el)
		fillTestAttributeValue(el)
		assert.EqualValues(t, testVal, el)
	}
}

func TestAnyValueArray_CopyTo(t *testing.T) {
	dest := NewAnyValueArray()
	// Test CopyTo to empty
	NewAnyValueArray().CopyTo(dest)
	assert.EqualValues(t, NewAnyValueArray(), dest)

	// Test CopyTo larger slice
	generateTestAnyValueArray().CopyTo(dest)
	assert.EqualValues(t, generateTestAnyValueArray(), dest)

	// Test CopyTo same size slice
	generateTestAnyValueArray().CopyTo(dest)
	assert.EqualValues(t, generateTestAnyValueArray(), dest)
}

func TestAnyValueArray_EnsureCapacity(t *testing.T) {
	es := generateTestAnyValueArray()
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
	assert.EqualValues(t, expectedEs, foundEs)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	oldLen := es.Len()
	assert.Equal(t, oldLen, len(expectedEs))
	es.EnsureCapacity(ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
}

func TestAnyValueArray_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestAnyValueArray()
	dest := NewAnyValueArray()
	src := generateTestAnyValueArray()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestAnyValueArray(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTestAnyValueArray(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestAnyValueArray().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestAnyValueArray_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewAnyValueArray()
	emptySlice.RemoveIf(func(el AttributeValue) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestAnyValueArray()
	pos := 0
	filtered.RemoveIf(func(el AttributeValue) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func generateTestInstrumentationLibrary() InstrumentationLibrary {
	tv := NewInstrumentationLibrary()
	fillTestInstrumentationLibrary(tv)
	return tv
}

func fillTestInstrumentationLibrary(tv InstrumentationLibrary) {
	tv.SetName("test_name")
	tv.SetVersion("test_version")
}

func generateTestAnyValueArray() AnyValueArray {
	tv := NewAnyValueArray()
	fillTestAnyValueArray(tv)
	return tv
}

func fillTestAnyValueArray(tv AnyValueArray) {
	l := 7
	tv.EnsureCapacity(l)
	for i := 0; i < l; i++ {
		fillTestAttributeValue(tv.AppendEmpty())
	}
}
