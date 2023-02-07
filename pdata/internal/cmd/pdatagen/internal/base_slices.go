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

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"bytes"
	"os"
)

const commonSliceTemplate = `// ${structName} logically represents a slice of ${elementName}.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewMutable${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	common${structName}
}

type Mutable${structName} struct {
	common${structName}
	preventConversion struct{} // nolint:unused
}

type common${structName} struct {
	orig *[]${originElementType}
}

func new${structName}FromOrig(orig *[]${originElementType}) ${structName} {
	return ${structName}{common${structName}{orig}}
}

func newMutable${structName}FromOrig(orig *[]${originElementType}) Mutable${structName} {
	return Mutable${structName}{common${structName}: common${structName}{orig}}
}

// NewMutable${structName} creates a ${structName} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewMutable${structName}() Mutable${structName} {
	orig := []${originElementType}(nil)
	return newMutable${structName}FromOrig(&orig)
}

func (es Mutable${structName}) AsImmutable() ${structName} {
	return ${structName}{common${structName}{orig: es.orig}}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewMutable${structName}()".
func (es common${structName}) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es ${structName}) At(i int) ${elementName} {
	return new${elementName}FromOrig(${origElementLink})
}

func (es Mutable${structName}) At(i int) Mutable${elementName} {
	return newMutable${elementName}FromOrig(${origElementLink})
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new ${structName} can be initialized:
//   es := NewMutable${structName}()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es Mutable${structName}) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]${originElementType}, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty ${elementName}.
// It returns the newly added ${elementName}.
func (es Mutable${structName}) AppendEmpty() Mutable${elementName} {
	*es.orig = append(*es.orig, ${emptyOriginElement})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es Mutable${structName}) MoveAndAppendTo(dest Mutable${structName}) {
	if *dest.orig == nil {
		// We can simply move the entire vector and avoid any allocations.
		*dest.orig = *es.orig
	} else {
		*dest.orig = append(*dest.orig, *es.orig...)
	}
	*es.orig = nil
}

// RemoveIf calls f sequentially for each element present in the slice.
// If f returns true, the element is removed from the slice.
func (es Mutable${structName}) RemoveIf(f func(Mutable${elementName}) bool) {
	newLen := 0
	for i := 0; i < len(*es.orig); i++ {
		if f(es.At(i)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*es.orig)[newLen] = (*es.orig)[i]
		newLen++
	}
	// TODO: Prevent memory leak by erasing truncated values.
	*es.orig = (*es.orig)[:newLen]
}`

const commonSliceTestTemplate = `func Test${structName}(t *testing.T) {
	es := NewMutable${structName}()
	assert.Equal(t, 0, es.Len())
	es = newMutable${structName}FromOrig(&[]${originElementType}{})
	assert.Equal(t, 0, es.Len())

	emptyVal := NewMutable${elementName}()
	testVal := generateTest${elementName}()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTest${elementName}(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func Test${structName}_CopyTo(t *testing.T) {
	dest := NewMutable${structName}()
	// Test CopyTo to empty
	NewMutable${structName}().CopyTo(dest)
	assert.Equal(t, NewMutable${structName}(), dest)

	// Test CopyTo larger slice
	generateTest${structName}().CopyTo(dest)
	assert.Equal(t, generateTest${structName}(), dest)

	// Test CopyTo same size slice
	generateTest${structName}().CopyTo(dest)
	assert.Equal(t, generateTest${structName}(), dest)
}

func Test${structName}_EnsureCapacity(t *testing.T) {
	es := generateTest${structName}()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTest${structName}(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTest${structName}().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTest${structName}(), es)
}

func Test${structName}_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTest${structName}()
	dest := NewMutable${structName}()
	src := generateTest${structName}()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTest${structName}(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTest${structName}(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTest${structName}().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func Test${structName}_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewMutable${structName}()
	emptySlice.RemoveIf(func(el Mutable${elementName}) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTest${structName}()
	pos := 0
	filtered.RemoveIf(func(el Mutable${elementName}) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}`

const slicePtrTemplate = `// CopyTo copies all elements from the current slice overriding the destination.
func (es common${structName}) CopyTo(dest Mutable${structName}) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			new${elementName}FromOrig((*es.orig)[i]).CopyTo(newMutable${elementName}FromOrig((*dest.orig)[i]))
		}
		return
	}
	origs := make([]${originName}, srcLen)
	wrappers := make([]*${originName}, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		new${elementName}FromOrig((*es.orig)[i]).CopyTo(newMutable${elementName}FromOrig(wrappers[i]))
	}
	*dest.orig = wrappers
}

// Sort sorts the ${elementName} elements within ${structName} given the
// provided less function so that two instances of ${structName}
// can be compared.
func (es Mutable${structName}) Sort(less func(a, b Mutable${elementName}) bool) {
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
}`

// TODO: Use assert.Less once https://github.com/stretchr/testify/pull/1339 is merged.
const slicePtrTestTemplate = `func Test${structName}_Sort(t *testing.T) {
	es := generateTest${structName}()
	es.Sort(func(a, b Mutable${elementName}) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) < uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b Mutable${elementName}) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) > uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}`

const sliceValueTemplate = `// CopyTo copies all elements from the current slice overriding the destination.
func (es common${structName}) CopyTo(dest Mutable${structName}) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
	} else {
		(*dest.orig) = make([]${originName}, srcLen)
	}

	for i := range *es.orig {
		new${elementName}FromOrig(&(*es.orig)[i]).CopyTo(newMutable${elementName}FromOrig(&(*dest.orig)[i]))
	}
}`

const commonSliceGenerateTest = `func generateTest${structName}() Mutable${structName} {
	es := NewMutable${structName}()
	fillTest${structName}(es)
	return es
}

func fillTest${structName}(es Mutable${structName}) {
	*es.orig = make([]${originElementType}, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = ${emptyOriginElement}
		fillTest${elementName}(newMutable${elementName}FromOrig(${origElementLink}))
	}
}`

type baseSlice interface {
	getName() string
	getPackageName() string
}

// sliceOfPtrs generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfPtrs struct {
	structName  string
	packageName string
	element     *messageValueStruct
}

func (ss *sliceOfPtrs) getName() string {
	return ss.structName
}

func (ss *sliceOfPtrs) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfPtrs) generateStruct(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(commonSliceTemplate, ss.templateFields()) + newLine + newLine)
	sb.WriteString(os.Expand(slicePtrTemplate, ss.templateFields()))
}

func (ss *sliceOfPtrs) generateTests(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(commonSliceTestTemplate, ss.templateFields()) + newLine + newLine)
	sb.WriteString(os.Expand(slicePtrTestTemplate, ss.templateFields()))
}

func (ss *sliceOfPtrs) generateTestValueHelpers(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(commonSliceGenerateTest, ss.templateFields()))
}

func (ss *sliceOfPtrs) templateFields() func(name string) string {
	return func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "elementName":
			return ss.element.structName
		case "originName":
			return ss.element.originFullName
		case "originElementType":
			return "*" + ss.element.originFullName
		case "emptyOriginElement":
			return "&" + ss.element.originFullName + "{}"
		case "origElementLink":
			return "(*es.orig)[i]"
		default:
			panic(name)
		}
	}
}

func (ss *sliceOfPtrs) generateAliases(_ *bytes.Buffer) {}

var _ baseStruct = (*sliceOfPtrs)(nil)

// sliceOfValues generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfValues struct {
	structName  string
	packageName string
	element     *messageValueStruct
}

func (ss *sliceOfValues) getName() string {
	return ss.structName
}

func (ss *sliceOfValues) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfValues) generateStruct(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(commonSliceTemplate, ss.templateFields()) + newLine + newLine)
	sb.WriteString(os.Expand(sliceValueTemplate, ss.templateFields()))
}

func (ss *sliceOfValues) generateTests(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(commonSliceTestTemplate, ss.templateFields()))
}

func (ss *sliceOfValues) generateTestValueHelpers(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(commonSliceGenerateTest, ss.templateFields()))
}

func (ss *sliceOfValues) templateFields() func(name string) string {
	return func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "elementName":
			return ss.element.structName
		case "originName":
			return ss.element.originFullName
		case "originElementType":
			return ss.element.originFullName
		case "emptyOriginElement":
			return ss.element.originFullName + "{}"
		case "origElementLink":
			return "&(*es.orig)[i]"
		default:
			panic(name)
		}
	}
}

func (ss *sliceOfValues) generateAliases(_ *bytes.Buffer) {}

var _ baseStruct = (*sliceOfValues)(nil)
