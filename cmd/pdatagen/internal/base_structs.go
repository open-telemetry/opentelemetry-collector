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

const messagePtrTemplate = `${description}
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

const messagePtrCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct to the dest.
func (ms ${structName}) CopyTo(dest ${structName}) {
	if ms.IsNil() {
		*dest.orig = nil
		return
	}
	if dest.IsNil() {
		dest.InitEmpty()
	}`

const messagePtrCopyToFooterTemplate = `}`

const messagePtrTestTemplate = `func Test${structName}_InitEmpty(t *testing.T) {
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

const messagePtrGenerateTestTemplate = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	tv.InitEmpty()
	fillTest${structName}(tv)
	return tv
}`

const messagePtrFillTestHeaderTemplate = `func fillTest${structName}(tv ${structName}) {`
const messagePtrFillTestFooterTemplate = `}`

const messageValueTemplate = `${description}
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	// orig points to the pointer ${originName} field contained somewhere else.
	orig *${originName}
}

func new${structName}(orig *${originName}) ${structName} {
	return ${structName}{orig: orig}
}

// New${structName} creates a new empty ${structName}.
//
// This must be used only in testing code since no "Set" method available.
func New${structName}() ${structName} {
	return new${structName}(&${originName}{})
}

// Deprecated: This function will be removed soon.
func (ms ${structName}) InitEmpty() {
	*ms.orig = ${originName}{}
}`

const messageValueCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct to the dest.
func (ms ${structName}) CopyTo(dest ${structName}) {`

const messageValueCopyToFooterTemplate = `}`

const messageValueTestTemplate = `
func Test${structName}_CopyTo(t *testing.T) {
	ms := New${structName}()
	generateTest${structName}().CopyTo(ms)
	assert.EqualValues(t, generateTest${structName}(), ms)
}`

const messageValueGenerateTestTemplate = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	fillTest${structName}(tv)
	return tv
}`

const messageValueFillTestHeaderTemplate = `func fillTest${structName}(tv ${structName}) {`
const messageValueFillTestFooterTemplate = `}`

const newLine = "\n"

type baseStruct interface {
	getName() string

	generateStruct(sb *strings.Builder)

	generateTests(sb *strings.Builder)

	generateTestValueHelpers(sb *strings.Builder)
}

type messagePtrStruct struct {
	structName     string
	description    string
	originFullName string
	fields         []baseField
}

func (ms *messagePtrStruct) getName() string {
	return ms.structName
}

func (ms *messagePtrStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(messagePtrTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messagePtrCopyToHeaderTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messagePtrCopyToFooterTemplate, func(name string) string {
		panic(name)
	}))
}

func (ms *messagePtrStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(messagePtrTestTemplate, func(name string) string {
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

func (ms *messagePtrStruct) generateTestValueHelpers(sb *strings.Builder) {
	sb.WriteString(os.Expand(messagePtrGenerateTestTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messagePtrFillTestHeaderTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messagePtrFillTestFooterTemplate, func(name string) string {
		panic(name)
	}))
}

var _ baseStruct = (*messagePtrStruct)(nil)

type messageValueStruct struct {
	structName     string
	description    string
	originFullName string
	fields         []baseField
}

func (ms *messageValueStruct) getName() string {
	return ms.structName
}

func (ms *messageValueStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageValueTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messageValueCopyToHeaderTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messageValueCopyToFooterTemplate, func(name string) string {
		panic(name)
	}))
}

func (ms *messageValueStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageValueTestTemplate, func(name string) string {
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

func (ms *messageValueStruct) generateTestValueHelpers(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageValueGenerateTestTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messageValueFillTestHeaderTemplate, func(name string) string {
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
	sb.WriteString(os.Expand(messageValueFillTestFooterTemplate, func(name string) string {
		panic(name)
	}))
}

var _ baseStruct = (*messageValueStruct)(nil)
