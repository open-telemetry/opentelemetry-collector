// Copyright 2020 OpenTelemetry Authors
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

package internal

import (
	"os"
	"strings"
)

const sliceTemplate = `// ${structName} logically represents a slice of ${elementName}.
//
// This is a reference type, if passsed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	orig *[]*${originName}
}

// New${structName} creates a ${structName} with "len" empty elements.
//
// es := New${structName}(4)
// for i := 0; i < es.Len(); i++ {
//     e := es.Get(i)
//     // Here should set all the values for e.
// }
func New${structName}(len int) ${structName} {
	if len == 0 {
		orig := []*${originName}(nil)
		return ${structName}{&orig}
	}
	// Slice for underlying orig.
	origs := make([]${originName}, len)
	// Slice for wrappers.
	wrappers := make([]*${originName}, len)
	for i := range origs {
		wrappers[i] = &origs[i]
	}
	return ${structName}{&wrappers}
}

func new${structName}(orig *[]*${originName}) ${structName} {
	return ${structName}{orig}
}

// Len returns the number of elements in the slice.
func (es ${structName}) Len() int {
	return len(*es.orig)
}

// Get returns the element associated with the given index.
//
// This function is used mostly for iterating over all the values in the slice:
// for i := 0; i < es.Len(); i++ {
//     e := es.Get(i)
//     ... // Do something with the element
// }
func (es ${structName}) Get(ix int) ${elementName} {
	return new${elementName}((*es.orig)[ix])
}

// Remove removes the element from the given index from the slice.
func (es ${structName}) Remove(ix int) {
	(*es.orig)[ix] = (*es.orig)[len(*es.orig)-1]
	*es.orig = (*es.orig)[:len(*es.orig)-1]
}

// Resize resizes the slice. This operation is equivalent with slice[to:from].
func (es ${structName}) Resize(from, to int) {
	*es.orig = (*es.orig)[from:to]
}`

const sliceTestTemplate = `func Test${structName}(t *testing.T) {
	es := New${structName}(0)
	assert.EqualValues(t, 0, es.Len())
	es = new${structName}(&[]*${originName}{})
	assert.EqualValues(t, 0, es.Len())
	es = New${structName}(13)
	defaultVal := New${elementName}()
	testVal := generateTest${elementName}()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, defaultVal, es.Get(i))
		fillTest${elementName}(es.Get(i))
		assert.EqualValues(t, testVal, es.Get(i))
	}

	// Test resize.
	const resizeLo = 2
	const resizeHi = 10
	expectedEs := make(map[*${originName}]bool, resizeHi-resizeLo)
	for i := resizeLo; i < resizeHi; i++ {
		expectedEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, resizeHi-resizeLo, len(expectedEs))
	es.Resize(resizeLo, resizeHi)
	assert.EqualValues(t, resizeHi-resizeLo, es.Len())
	foundEs := make(map[*${originName}]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test remove.
	const removePos = 2
	delete(expectedEs, es.Get(removePos).orig)
	es.Remove(removePos)
	assert.EqualValues(t, resizeHi-resizeLo-1, es.Len())
	foundEs = make(map[*${originName}]bool, resizeHi-resizeLo)
	for i := 0; i < es.Len(); i++ {
		foundEs[es.Get(i).orig] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
}`

const sliceGenerateTest = `func generateTest${structName}() ${structName} {
	tv := New${structName}(13)
	for i := 0; i < tv.Len(); i++ {
		fillTest${elementName}(tv.Get(i))
	}
	return tv
}`

const messageTemplate = `${description}
//
// This is a reference type, if passsed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	// Wrap OTLP ${originName}.
	orig *${originName}
}

// New${structName} creates a new empty ${structName}.
func New${structName}() ${structName} {
	return ${structName}{&${originName}{}}
}

func new${structName}(orig *${originName}) ${structName} {
	return ${structName}{orig}
}`

const messageTestHeaderTemplate = `func Test${structName}(t *testing.T) {
	ms := New${structName}()
	assert.EqualValues(t, new${structName}(&${originName}{}), ms)`

const messageTestFooterTemplate = `	assert.EqualValues(t, generateTest${structName}(), ms)
}`

const messageGenerateTestHeaderTemplate = `func generateTest${structName}() ${structName} {
	tv := New${structName}()`
const messageGenerateTestFooterTemplate = `	return tv
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
	ss.element.generateFillTest = true
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
	structName       string
	description      string
	originFullName   string
	fields           []baseField
	generateFillTest bool
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
}

func (ms *messageStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageTestHeaderTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		default:
			panic(name)
		}
	}))
	// Write accessors tests for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine + newLine)
		f.generateAccessorsTests(ms, sb)
	}
	sb.WriteString(newLine + newLine)
	sb.WriteString(os.Expand(messageTestFooterTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		default:
			panic(name)
		}
	}))
}

func (ms *messageStruct) generateTestValueHelpers(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageGenerateTestHeaderTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messageGenerateTestFooterTemplate, func(name string) string {
		switch name {
		default:
			panic(name)
		}
	}))

	if ms.generateFillTest {
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
			switch name {
			default:
				panic(name)
			}
		}))
	}
}

var _ baseStruct = (*messageStruct)(nil)
