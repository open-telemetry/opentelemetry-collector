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
	"os"
	"strings"
)

const sliceTemplate = `// ${structName} logically represents a slice of ${elementName}.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	// orig points to the slice ${originName} field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like Resize.
	orig *[]*${originName}
}

func new${structName}(orig *[]*${originName}) ${structName} {
	return ${structName}{orig}
}

// New${structName} creates a ${structName} with 0 elements.
// Can use "Resize" to initialize with a given length.
func New${structName}() ${structName} {
	orig := []*${originName}(nil)
	return ${structName}{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New${structName}()".
func (es ${structName}) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     ... // Do something with the element
// }
func (es ${structName}) At(ix int) ${elementName} {
	return new${elementName}(&(*es.orig)[ix])
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es ${structName}) MoveAndAppendTo(dest ${structName}) {
	if es.Len() == 0 {
		// Just to ensure that we always return a Slice with nil elements.
		*es.orig = nil
		return
	}
	if dest.Len() == 0 {
		*dest.orig = *es.orig
		*es.orig = nil
		return
	}
	*dest.orig = append(*dest.orig, *es.orig...)
	*es.orig = nil
	return
}

// CopyTo copies all elements from the current slice to the dest.
func (es ${structName}) CopyTo(dest ${structName}) {
	newLen := es.Len()
	if newLen == 0 {
		*dest.orig = []*${originName}(nil)
		return
	}
	oldLen := dest.Len()
	if newLen <= oldLen {
		(*dest.orig) = (*dest.orig)[:newLen]
		for i, el := range *es.orig {
			new${elementName}(&el).CopyTo(new${elementName}(&(*dest.orig)[i]))
		}
		return
	}
	origs := make([]${originName}, newLen)
	wrappers := make([]*${originName}, newLen)
	for i, el := range *es.orig {
		wrappers[i] = &origs[i]
		new${elementName}(&el).CopyTo(new${elementName}(&wrappers[i]))
	}
	*dest.orig = wrappers
}

// Resize is an operation that resizes the slice:
// 1. If newLen is 0 then the slice is replaced with a nil slice.
// 2. If the newLen <= len then equivalent with slice[0:newLen].
// 3. If the newLen > len then (newLen - len) empty elements will be appended to the slice.
//
// Here is how a new ${structName} can be initialized:
// es := New${structName}()
// es.Resize(4)
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     // Here should set all the values for e.
// }
func (es ${structName}) Resize(newLen int) {
	if newLen == 0 {
		(*es.orig) = []*${originName}(nil)
		return
	}
	oldLen := len(*es.orig)
	if newLen <= oldLen {
		(*es.orig) = (*es.orig)[:newLen]
		return
	}
	// TODO: Benchmark and optimize this logic.
	extraOrigs := make([]${originName}, newLen-oldLen)
	oldOrig := (*es.orig)
	for i := range extraOrigs {
		oldOrig = append(oldOrig, &extraOrigs[i])
	}
	(*es.orig) = oldOrig
}

// Append will increase the length of the ${structName} by one and set the
// given ${elementName} at that new position.  The original ${elementName}
// could still be referenced so do not reuse it after passing it to this
// method.
func (es ${structName}) Append(e ${elementName}) {
	*es.orig = append(*es.orig, *e.orig)
}`

const sliceTestTemplate = `func Test${structName}(t *testing.T) {
	es := New${structName}()
	assert.EqualValues(t, 0, es.Len())
	es = new${structName}(&[]*${originName}{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := New${elementName}()
	emptyVal.InitEmpty()
	testVal := generateTest${elementName}()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTest${elementName}(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func Test${structName}_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTest${structName}()
	dest := New${structName}()
	src := generateTest${structName}()
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTest${structName}(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.EqualValues(t, generateTest${structName}(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTest${structName}().MoveAndAppendTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func Test${structName}_CopyTo(t *testing.T) {
	dest := New${structName}()
	// Test CopyTo to empty
	New${structName}().CopyTo(dest)
	assert.EqualValues(t, New${structName}(), dest)

	// Test CopyTo larger slice
	generateTest${structName}().CopyTo(dest)
	assert.EqualValues(t, generateTest${structName}(), dest)

	// Test CopyTo same size slice
	generateTest${structName}().CopyTo(dest)
	assert.EqualValues(t, generateTest${structName}(), dest)
}

func Test${structName}_Resize(t *testing.T) {
	es := generateTest${structName}()
	emptyVal := New${elementName}()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*${originName}]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*${originName}]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, New${structName}(), es)
}

func Test${structName}_Append(t *testing.T) {
	es := generateTest${structName}()
	emptyVal := New${elementName}()
	emptyVal.InitEmpty()

	es.Append(emptyVal)
	assert.EqualValues(t, *(es.At(7)).orig, *emptyVal.orig)

	emptyVal2 := New${elementName}()
	emptyVal2.InitEmpty()

	es.Append(emptyVal2)
	assert.EqualValues(t, *(es.At(8)).orig, *emptyVal2.orig)

	assert.Equal(t, 9, es.Len())
}`

const sliceGenerateTest = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	fillTest${structName}(tv)
	return tv
}

func fillTest${structName}(tv ${structName}) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTest${elementName}(tv.At(i))
	}
}`

const messageTemplate = `${description}
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	// orig points to the pointer ${originName} field contained somewhere else.
	// We use pointer-to-pointer to be able to modify it in InitEmpty func.
	orig **${originName}
}

func new${structName}(orig **${originName}) ${structName} {
	return ${structName}{orig}
}

// New${structName} creates a new "nil" ${structName}.
// To initialize the struct call "InitEmpty".
//
// This must be used only in testing code since no "Set" method available.
func New${structName}() ${structName} {
	orig := (*${originName})(nil)
	return new${structName}(&orig)
}

// InitEmpty overwrites the current value with empty.
func (ms ${structName}) InitEmpty() {
	*ms.orig = &${originName}{}
}

// IsNil returns true if the underlying data are nil.
//
// Important: All other functions will cause a runtime error if this returns "true".
func (ms ${structName}) IsNil() bool {
	return *ms.orig == nil
}`

const messageCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct to the dest.
func (ms ${structName}) CopyTo(dest ${structName}) {
	if ms.IsNil() {
		*dest.orig = nil
		return
	}
	if dest.IsNil() {
		dest.InitEmpty()
	}`

const messageCopyToFooterTemplate = `}`

const messageTestTemplate = `func Test${structName}_InitEmpty(t *testing.T) {
	ms := New${structName}()
	assert.True(t, ms.IsNil())
	ms.InitEmpty()
	assert.False(t, ms.IsNil())
}

func Test${structName}_CopyTo(t *testing.T) {
	ms := New${structName}()
	New${structName}().CopyTo(ms)
	assert.True(t, ms.IsNil())
	generateTest${structName}().CopyTo(ms)
	assert.EqualValues(t, generateTest${structName}(), ms)
}`

const messageGenerateTestTemplate = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	tv.InitEmpty()
	fillTest${structName}(tv)
	return tv
}`

const messageFillTestHeaderTemplate = `func fillTest${structName}(tv ${structName}) {`
const messageFillTestFooterTemplate = `}`

const newLine = "\n"

type baseStruct interface {
	generateStruct(sb *strings.Builder)

	generateTests(sb *strings.Builder)

	generateTestValueHelpers(sb *strings.Builder)
}

// Will generate code only for the slice struct.
type sliceStruct struct {
	structName string
	element    *messageStruct
}

func (ss *sliceStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(sliceTemplate, func(name string) string {
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
	}))
}

func (ss *sliceStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(sliceTestTemplate, func(name string) string {
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
	}))
}

func (ss *sliceStruct) generateTestValueHelpers(sb *strings.Builder) {
	sb.WriteString(os.Expand(sliceGenerateTest, func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "elementName":
			return ss.element.structName
		default:
			panic(name)
		}
	}))
}

var _ baseStruct = (*sliceStruct)(nil)

type messageStruct struct {
	structName     string
	description    string
	originFullName string
	fields         []baseField
}

func (ms *messageStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		case "description":
			return ms.description
		default:
			panic(name)
		}
	}))
	// Write accessors for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine + newLine)
		f.generateAccessors(ms, sb)
	}
	sb.WriteString(newLine + newLine)
	sb.WriteString(os.Expand(messageCopyToHeaderTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		default:
			panic(name)
		}
	}))
	// Write accessors CopyTo for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine)
		f.generateCopyToValue(sb)
	}
	sb.WriteString(newLine)
	sb.WriteString(os.Expand(messageCopyToFooterTemplate, func(name string) string {
		panic(name)
	}))
}

func (ms *messageStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		default:
			panic(name)
		}
	}))
	// Write accessors tests for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine + newLine)
		f.generateAccessorsTest(ms, sb)
	}
}

func (ms *messageStruct) generateTestValueHelpers(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageGenerateTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		default:
			panic(name)
		}
	}))
	sb.WriteString(newLine + newLine)
	sb.WriteString(os.Expand(messageFillTestHeaderTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		default:
			panic(name)
		}
	}))
	// Write accessors test value for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine)
		f.generateSetWithTestValue(sb)
	}
	sb.WriteString(newLine)
	sb.WriteString(os.Expand(messageFillTestFooterTemplate, func(name string) string {
		panic(name)
	}))
}

var _ baseStruct = (*messageStruct)(nil)
