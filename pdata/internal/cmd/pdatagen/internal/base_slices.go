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
func (es mutable${structName}) MoveAndAppendTo(dest mutable${structName}) {
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
func (es mutable${structName}) RemoveIf(f func(Mutable${elementName}) bool) {
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
	*es.orig = (*es.orig)[:newLen]
}`

const commonSliceTestTemplate = `

func Test${structName}_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTest${structName}()
	dest := New${structName}()
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
	emptySlice := New${structName}()
	emptySlice.RemoveIf(func(el ${elementName}) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTest${structName}()
	pos := 0
	filtered.RemoveIf(func(el ${elementName}) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}`

const slicePtrTemplate = `// ${structName} logically represents a slice of ${elementName}.
type ${structName} interface {
	common${structName}
	At(ix int) ${elementName}
}

type Mutable${structName} interface {
	common${structName}
	At(ix int) Mutable${elementName}
	EnsureCapacity(newCap int)
	AppendEmpty() Mutable${elementName}
	Sort(less func(a, b Mutable${elementName}) bool)
}

type common${structName} interface {
	Len() int
	CopyTo(dest Mutable${structName})
	getOrig() *[]*${originName}
}

type immutable${structName} struct {
	orig *[]*${originName}
}

type mutable${structName} struct {
	immutable${structName}
}

func (es immutable${structName}) getOrig() *[]*${originName} {
	return es.orig
}

func newImmutable${structName}(orig *[]*${originName}) immutable${structName} {
	return immutable${structName}{orig}
}

func newMutable${structName}(orig *[]*${originName}) mutable${structName} {
	return mutable${structName}{immutable${structName}{orig}}
}

// New${structName} creates a ${structName} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func New${structName}() Mutable${structName} {
	orig := []*${originName}(nil)
	return newMutable${structName}(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New${structName}()".
func (es immutable${structName}) Len() int {
	return len(*es.getOrig())
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es immutable${structName}) At(ix int) ${elementName} {
	return newImmutable${elementName}((*es.getOrig())[ix])
}

func (es mutable${structName}) At(ix int) Mutable${elementName} {
	return newMutable${elementName}((*es.getOrig())[ix])
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es immutable${structName}) CopyTo(dest Mutable${structName}) {
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]
		for i := range *es.getOrig() {
			newImmutable${elementName}((*es.getOrig())[i]).CopyTo(newMutable${elementName}((*dest.getOrig())[i]))
		}
		return
	}
	origs := make([]${originName}, srcLen)
	wrappers := make([]*${originName}, srcLen)
	for i := range *es.getOrig() {
		wrappers[i] = &origs[i]
		newImmutable${elementName}((*es.getOrig())[i]).CopyTo(newMutable${elementName}(wrappers[i]))
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
func (es mutable${structName}) EnsureCapacity(newCap int) {
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
func (es mutable${structName}) AppendEmpty() Mutable${elementName} {
	*es.getOrig() = append(*es.getOrig(), &${originName}{})
	return es.At(es.Len() - 1)
}

// Sort sorts the ${elementName} elements within ${structName} given the
// provided less function so that two instances of ${structName}
// can be compared.
func (es mutable${structName}) Sort(less func(a, b Mutable${elementName}) bool) {
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
	testVal := generateTest${elementName}()
	assert.Equal(t, 7, cap(*es.orig))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, el)
		fillTest${elementName}(el)
		assert.Equal(t, testVal, el)
	}
}

func Test${structName}_CopyTo(t *testing.T) {
	dest := New${structName}()
	// Test CopyTo to empty
	New${structName}().CopyTo(dest)
	assert.Equal(t, New${structName}(), dest)

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
	expectedEs := make(map[*${originName}]bool)
	for i := 0; i < es.Len(); i++ {
		expectedEs[es.At(i).orig] = true
	}
	assert.Equal(t, es.Len(), len(expectedEs))
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	foundEs := make(map[*${originName}]bool, es.Len())
	for i := 0; i < es.Len(); i++ {
		foundEs[es.At(i).orig] = true
	}
	assert.Equal(t, expectedEs, foundEs)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	oldLen := es.Len()
	expectedEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[es.At(i).orig] = true
	}
	assert.Equal(t, oldLen, len(expectedEs))
	es.EnsureCapacity(ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	foundEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[es.At(i).orig] = true
	}
	assert.Equal(t, expectedEs, foundEs)
}`

const slicePtrGenerateTest = `

func generateTest${structName}() Mutable${structName} {
	tv := New${structName}()
	fillTest${structName}(tv)
	return tv
}

func fillTest${structName}(tv Mutable${structName}) {
	*tv.orig = make([]*${originName}, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &${originName}{}
		fillTest${elementName}(new${elementName}((*tv.orig)[i]))
	}
}`

const sliceValueTemplate = `// ${structName} logically represents a slice of ${elementName}.
type ${structName} interface {
	common${structName}
	At(ix int) ${elementName}
}

type Mutable${structName} interface {
	common${structName}
	RemoveIf(f func(Mutable${elementName}) bool)
	At(ix int) Mutable${elementName}
	EnsureCapacity(newCap int)
	AppendEmpty() Mutable${elementName}
}

type common${structName} interface {
	Len() int
	CopyTo(dest Mutable${structName})
	getOrig() *[]${originName}
}

type immutable${structName} struct {
	orig *[]${originName}
}

type mutable${structName} struct {
	immutable${structName}
}

func newImmutable${structName}(orig *[]${originName}) immutable${structName} {
	return immutable${structName}{orig}
}

func newMutable${structName}(orig *[]${originName}) mutable${structName} {
	return mutable${structName}{immutable${structName}{orig}}
}

func (es immutable${structName}) getOrig() *[]${originName} {
	return es.orig
}

// New${structName} creates a ${structName} with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func New${structName}() Mutable${structName} {
	orig := []${originName}(nil)
	return newMutable${structName}(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New${structName}()".
func (es immutable${structName}) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es immutable${structName}) At(ix int) ${elementName} {
	return newImmutable${elementName}(&(*es.orig)[ix])
}

func (es mutable${structName}) At(ix int) Mutable${elementName} {
	return newMutable${elementName}(&(*es.getOrig())[ix])
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es immutable${structName}) CopyTo(dest Mutable${structName}) {
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]
	} else {
		(*dest.getOrig()) = make([]${originName}, srcLen)
	}

	for i := range *es.getOrig() {
		newImmutable${elementName}(&(*es.getOrig())[i]).CopyTo(newMutable${elementName}(&(*dest.getOrig())[i]))
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
func (es mutable${structName}) EnsureCapacity(newCap int) {
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
func (es mutable${structName}) AppendEmpty() Mutable${elementName} {
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
	testVal := generateTest${elementName}()
	assert.Equal(t, 7, cap(*es.orig))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, el)
		fillTest${elementName}(el)
		assert.Equal(t, testVal, el)
	}
}

func Test${structName}_CopyTo(t *testing.T) {
	dest := New${structName}()
	// Test CopyTo to empty
	New${structName}().CopyTo(dest)
	assert.Equal(t, New${structName}(), dest)

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
	expectedEs := make(map[*${originName}]bool)
	for i := 0; i < es.Len(); i++ {
		expectedEs[es.At(i).orig] = true
	}
	assert.Equal(t, es.Len(), len(expectedEs))
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	foundEs := make(map[*${originName}]bool, es.Len())
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
}`

const sliceValueGenerateTest = `func generateTest${structName}() Mutable${structName} {
	tv := New${structName}()
	fillTest${structName}(tv)
	return tv
}

func fillTest${structName}(tv ${structName}) {
	*tv.orig = make([]${originName}, 7)
	for i := 0; i < 7; i++ {
		fillTest${elementName}(new${elementName}(&(*tv.orig)[i]))
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

func (ss *sliceOfValues) generateAliases(_ *bytes.Buffer) {}

var _ baseStruct = (*sliceOfValues)(nil)
