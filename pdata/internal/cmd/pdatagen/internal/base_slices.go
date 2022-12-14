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

const commonSliceTemplate = `
// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es ${structName}) MoveAndAppendTo(dest ${structName}) {
	if *dest.getOrig() == nil {
		// We can simply move the entire vector and avoid any allocations.
		*dest.getOrig() = *es.getOrig()
	} else {
		*dest.getOrig() = append(*dest.getOrig(), *es.getOrig()...)
	}
	*es.getOrig() = nil
}

// RemoveIf calls f sequentially for each element present in the slice.
// If f returns true, the element is removed from the slice.
func (es ${structName}) RemoveIf(f func(${elementName}) bool) {
	newLen := 0
	for i := 0; i < len(*es.getOrig()); i++ {
		if f(es.At(i)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*es.getOrig())[newLen] = (*es.getOrig())[i]
		newLen++
	}
	// TODO: Prevent memory leak by erasing truncated values.
	*es.getOrig() = (*es.getOrig())[:newLen]
}`

const commonSliceTestTemplate = `

func Test${structName}_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := ${structName}(internal.GenerateTest${structName}())
	dest := New${structName}()
	src := ${structName}(internal.GenerateTest${structName}())
	src.MoveAndAppendTo(dest)
	assert.Equal(t, ${structName}(internal.GenerateTest${structName}()), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, ${structName}(internal.GenerateTest${structName}()), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	${structName}(internal.GenerateTest${structName}()).MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func Test${structName}_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := New${structName}()
	emptySlice.RemoveIf(func(el ${elementName}) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := ${structName}(internal.GenerateTest${structName}())
	pos := 0
	filtered.RemoveIf(func(el ${elementName}) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}`

const slicePtrTemplate = `// ${structName} logically represents a slice of ${elementName}.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} internal.${structName}

func new${structName}(orig *[]*${originName}) ${structName} {
	return ${structName}(internal.New${structName}(orig))
}

func (ms ${structName}) getOrig() *[]*${originName} {
	return internal.GetOrig${structName}(internal.${structName}(ms))
}

// New${structName} creates a ${structName} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func New${structName}() ${structName} {
	orig := []*${originName}(nil)
	return new${structName}(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New${structName}()".
func (es ${structName}) Len() int {
	return len(*es.getOrig())
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es ${structName}) At(ix int) ${elementName} {
	return new${elementName}((*es.getOrig())[ix])
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es ${structName}) CopyTo(dest ${structName}) {
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]
		for i := range *es.getOrig() {
			new${elementName}((*es.getOrig())[i]).CopyTo(new${elementName}((*dest.getOrig())[i]))
		}
		return
	}
	origs := make([]${originName}, srcLen)
	wrappers := make([]*${originName}, srcLen)
	for i := range *es.getOrig() {
		wrappers[i] = &origs[i]
		new${elementName}((*es.getOrig())[i]).CopyTo(new${elementName}(wrappers[i]))
	}
	*dest.getOrig() = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new ${structName} can be initialized:
//   es := New${structName}()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es ${structName}) EnsureCapacity(newCap int) {
	oldCap := cap(*es.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*${originName}, len(*es.getOrig()), newCap)
	copy(newOrig, *es.getOrig())
	*es.getOrig() = newOrig
}

// AppendEmpty will append to the end of the slice an empty ${elementName}.
// It returns the newly added ${elementName}.
func (es ${structName}) AppendEmpty() ${elementName} {
	*es.getOrig() = append(*es.getOrig(), &${originName}{})
	return es.At(es.Len() - 1)
}

// Sort sorts the ${elementName} elements within ${structName} given the
// provided less function so that two instances of ${structName}
// can be compared.
func (es ${structName}) Sort(less func(a, b ${elementName}) bool) {
	sort.SliceStable(*es.getOrig(), func(i, j int) bool { return less(es.At(i), es.At(j)) })
}
`

const slicePtrTestTemplate = `func Test${structName}(t *testing.T) {
	es := New${structName}()
	assert.Equal(t, 0, es.Len())
	es = new${structName}(&[]*${originName}{})
	assert.Equal(t, 0, es.Len())

	es.EnsureCapacity(7)
	emptyVal := new${elementName}(&${originName}{})
	testVal := ${elementName}(internal.GenerateTest${elementName}())
	assert.Equal(t, 7, cap(*es.getOrig()))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, el)
		internal.FillTest${elementName}(internal.${elementName}(el))
		assert.Equal(t, testVal, el)
	}
}

func Test${structName}_CopyTo(t *testing.T) {
	dest := New${structName}()
	// Test CopyTo to empty
	New${structName}().CopyTo(dest)
	assert.Equal(t, New${structName}(), dest)

	// Test CopyTo larger slice
	${structName}(internal.GenerateTest${structName}()).CopyTo(dest)
	assert.Equal(t, ${structName}(internal.GenerateTest${structName}()), dest)

	// Test CopyTo same size slice
	${structName}(internal.GenerateTest${structName}()).CopyTo(dest)
	assert.Equal(t, ${structName}(internal.GenerateTest${structName}()), dest)
}

func Test${structName}_EnsureCapacity(t *testing.T) {
	es := ${structName}(internal.GenerateTest${structName}())
	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	expectedEs := make(map[*${originName}]bool)
	for i := 0; i < es.Len(); i++ {
		expectedEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, es.Len(), len(expectedEs))
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	foundEs := make(map[*${originName}]bool, es.Len())
	for i := 0; i < es.Len(); i++ {
		foundEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, expectedEs, foundEs)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	oldLen := es.Len()
	expectedEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, oldLen, len(expectedEs))
	es.EnsureCapacity(ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.getOrig()))
	foundEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, expectedEs, foundEs)
}`

const slicePtrGenerateTest = `func GenerateTest${structName}() ${structName} {
	orig := []*${originName}{}
	tv := New${structName}(&orig)
	FillTest${structName}(tv)
	return tv
}

func FillTest${structName}(tv ${structName}) {
	*tv.orig = make([]*${originName}, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &${originName}{}
		FillTest${elementName}(New${elementName}((*tv.orig)[i]))
	}
}`

const slicePtrInternalTemplate = `
type ${structName} struct {
	orig *[]*${originName}
}

func GetOrig${structName}(ms ${structName}) *[]*${originName} {
	return ms.orig
}

func New${structName}(orig *[]*${originName}) ${structName} {
	return ${structName}{orig: orig}
}`

const sliceValueTemplate = `// ${structName} logically represents a slice of ${elementName}.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} internal.${structName}

func new${structName}(orig *[]${originName}) ${structName} {
	return ${structName}(internal.New${structName}(orig))
}

func (ms ${structName}) getOrig() *[]${originName} {
	return internal.GetOrig${structName}(internal.${structName}(ms))
}

// New${structName} creates a ${structName} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func New${structName}() ${structName} {
	orig := []${originName}(nil)
	return ${structName}(internal.New${structName}(&orig))
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New${structName}()".
func (es ${structName}) Len() int {
	return len(*es.getOrig())
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es ${structName}) At(ix int) ${elementName} {
	return new${elementName}(&(*es.getOrig())[ix])
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es ${structName}) CopyTo(dest ${structName}) {
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]
	} else {
		(*dest.getOrig()) = make([]${originName}, srcLen)
	}

	for i := range *es.getOrig() {
		new${elementName}(&(*es.getOrig())[i]).CopyTo(new${elementName}(&(*dest.getOrig())[i]))
	}
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new ${structName} can be initialized:
//   es := New${structName}()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es ${structName}) EnsureCapacity(newCap int) {
	oldCap := cap(*es.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]${originName}, len(*es.getOrig()), newCap)
	copy(newOrig, *es.getOrig())
	*es.getOrig() = newOrig
}

// AppendEmpty will append to the end of the slice an empty ${elementName}.
// It returns the newly added ${elementName}.
func (es ${structName}) AppendEmpty() ${elementName} {
	*es.getOrig() = append(*es.getOrig(), ${originName}{})
	return es.At(es.Len() - 1)
}`

const sliceValueTestTemplate = `func Test${structName}(t *testing.T) {
	es := New${structName}()
	assert.Equal(t, 0, es.Len())
	es = new${structName}(&[]${originName}{})
	assert.Equal(t, 0, es.Len())

	es.EnsureCapacity(7)
	emptyVal := new${elementName}(&${originName}{})
	testVal := ${elementName}(internal.GenerateTest${elementName}())
	assert.Equal(t, 7, cap(*es.getOrig()))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, el)
		internal.FillTest${elementName}(internal.${elementName}(el))
		assert.Equal(t, testVal, el)
	}
}

func Test${structName}_CopyTo(t *testing.T) {
	dest := New${structName}()
	// Test CopyTo to empty
	New${structName}().CopyTo(dest)
	assert.Equal(t, New${structName}(), dest)

	// Test CopyTo larger slice
	${structName}(internal.GenerateTest${structName}()).CopyTo(dest)
	assert.Equal(t, ${structName}(internal.GenerateTest${structName}()), dest)

	// Test CopyTo same size slice
	${structName}(internal.GenerateTest${structName}()).CopyTo(dest)
	assert.Equal(t, ${structName}(internal.GenerateTest${structName}()), dest)
}

func Test${structName}_EnsureCapacity(t *testing.T) {
	es := ${structName}(internal.GenerateTest${structName}())
	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	expectedEs := make(map[*${originName}]bool)
	for i := 0; i < es.Len(); i++ {
		expectedEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, es.Len(), len(expectedEs))
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	foundEs := make(map[*${originName}]bool, es.Len())
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
}`

const sliceValueGenerateTest = `func GenerateTest${structName}() ${structName} {
	orig := []${originName}{}
	tv := New${structName}(&orig)
	FillTest${structName}(tv)
	return tv
}

func FillTest${structName}(tv ${structName}) {
	*tv.orig = make([]${originName}, 7)
	for i := 0; i < 7; i++ {
		FillTest${elementName}(New${elementName}(&(*tv.orig)[i]))
	}
}`

const sliceValueInternalTemplate = `
type ${structName} struct {
	orig *[]${originName}
}

func GetOrig${structName}(ms ${structName}) *[]${originName} {
	return ms.orig
}

func New${structName}(orig *[]${originName}) ${structName} {
	return ${structName}{orig: orig}
}`

type baseSlice interface {
	getName() string
	getPackageName() string
}

// Will generate code only for a slice of pointer fields.
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
	sb.WriteString(os.Expand(slicePtrTemplate, ss.templateFields()))
	sb.WriteString(os.Expand(commonSliceTemplate, ss.templateFields()))
}

func (ss *sliceOfPtrs) generateTests(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(slicePtrTestTemplate, ss.templateFields()))
	sb.WriteString(os.Expand(commonSliceTestTemplate, ss.templateFields()))
}

func (ss *sliceOfPtrs) generateTestValueHelpers(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(slicePtrGenerateTest, ss.templateFields()))
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
		default:
			panic(name)
		}
	}
}

func (ss *sliceOfPtrs) generateInternal(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(slicePtrInternalTemplate, func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "originName":
			return ss.element.originFullName
		default:
			panic(name)
		}
	}))
	sb.WriteString(newLine + newLine)
}

var _ baseStruct = (*sliceOfPtrs)(nil)

// Will generate code only for a slice of value fields.
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
	sb.WriteString(os.Expand(sliceValueTemplate, ss.templateFields()))
	sb.WriteString(os.Expand(commonSliceTemplate, ss.templateFields()))
}

func (ss *sliceOfValues) generateTests(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(sliceValueTestTemplate, ss.templateFields()))
	sb.WriteString(os.Expand(commonSliceTestTemplate, ss.templateFields()))
}

func (ss *sliceOfValues) generateTestValueHelpers(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(sliceValueGenerateTest, ss.templateFields()))
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
		default:
			panic(name)
		}
	}
}

func (ss *sliceOfValues) generateInternal(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(sliceValueInternalTemplate, func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "originName":
			return ss.element.originFullName
		default:
			panic(name)
		}
	}))
	sb.WriteString(newLine + newLine)
}

var _ baseStruct = (*sliceOfValues)(nil)
