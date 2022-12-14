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

const messageValueTemplate = `${description}
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.

type ${structName} internal.${internalStructName}

func new${structName}(orig *${originName}) ${structName} {
	return ${structName}(internal.New${internalStructName}(orig))
}

func (ms ${structName}) getOrig() *${originName} {
	return internal.GetOrig${internalStructName}(internal.${internalStructName}(ms))
}

// New${structName} creates a new empty ${structName}.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func New${structName}() ${structName} {
	return new${structName}(&${originName}{})
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms ${structName}) MoveTo(dest ${structName}) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = ${originName}{}
}`

const messageValueCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct overriding the destination.
func (ms ${structName}) CopyTo(dest ${structName}) {`

const messageValueCopyToFooterTemplate = `}`

const messageValueTestTemplate = `
func Test${structName}_MoveTo(t *testing.T) {
	ms := ${structName}(internal.GenerateTest${internalStructName}())
	dest := New${structName}()
	ms.MoveTo(dest)
	assert.Equal(t, New${structName}(), ms)
	assert.Equal(t, ${structName}(internal.GenerateTest${internalStructName}()), dest)
}

func Test${structName}_CopyTo(t *testing.T) {
	ms := New${structName}()
	orig := New${structName}()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = ${structName}(internal.GenerateTest${internalStructName}())
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}`

const messageValueGenerateTestTemplate = `func GenerateTest${internalStructName}() ${internalStructName} {
	orig := ${originName}{}
	tv := New${internalStructName}(&orig)
	FillTest${internalStructName}(tv)
	return tv
}`

const messageValueFillTestHeaderTemplate = `func FillTest${internalStructName}(tv ${internalStructName}) {`
const messageValueFillTestFooterTemplate = `}`

const messageValueAliasTemplate = `
type ${internalStructName} struct {
	orig *${originName}
}

func GetOrig${internalStructName}(ms ${internalStructName}) *${originName} {
	return ms.orig
}

func New${internalStructName}(orig *${originName}) ${internalStructName} {
	return ${internalStructName}{orig: orig}
}`

const newLine = "\n"

type baseStruct interface {
	getName() string

	getPackageName() string

	generateStruct(sb *bytes.Buffer)

	generateTests(sb *bytes.Buffer)

	generateTestValueHelpers(sb *bytes.Buffer)

	generateInternal(sb *bytes.Buffer)
}

type messageValueStruct struct {
	structName         string
	internalStructName string
	packageName        string
	description        string
	originFullName     string
	fields             []baseField
}

func (ms *messageValueStruct) getName() string {
	return ms.structName
}

func (ms *messageValueStruct) getInternalName() string {
	if ms.internalStructName != "" {
		return ms.internalStructName
	}
	return ms.structName
}

func (ms *messageValueStruct) getPackageName() string {
	return ms.packageName
}

func (ms *messageValueStruct) generateStruct(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "internalStructName":
			return ms.getInternalName()
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
		f.generateCopyToValue(ms, sb)
	}
	sb.WriteString(newLine)
	sb.WriteString(os.Expand(messageValueCopyToFooterTemplate, func(name string) string {
		panic(name)
	}))
}

func (ms *messageValueStruct) generateTests(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "internalStructName":
			return ms.getInternalName()
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

func (ms *messageValueStruct) generateTestValueHelpers(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueGenerateTestTemplate, func(name string) string {
		switch name {
		case "internalStructName":
			return ms.getInternalName()
		case "originName":
			return ms.originFullName
		default:
			panic(name)
		}
	}))
	sb.WriteString(newLine + newLine)
	sb.WriteString(os.Expand(messageValueFillTestHeaderTemplate, func(name string) string {
		switch name {
		case "internalStructName":
			return ms.getInternalName()
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

func (ms *messageValueStruct) generateInternal(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueAliasTemplate, func(name string) string {
		switch name {
		case "internalStructName":
			return ms.getInternalName()
		case "originName":
			return ms.originFullName
		default:
			panic(name)
		}
	}))
	sb.WriteString(newLine + newLine)
}

var _ baseStruct = (*messageValueStruct)(nil)
