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
	"os"
	"strings"
)

const messageValueTemplate = `${description}
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	orig *${originName}
}

func new${structName}(orig *${originName}) ${structName} {
	return ${structName}{orig: orig}
}

// New${structName} creates a new empty ${structName}.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func New${structName}() ${structName} {
	return new${structName}(&${originName}{})
}

// MoveTo moves all properties from the current struct to dest
// resetting the current instance to its zero value
func (ms ${structName}) MoveTo(dest ${structName}) {
	*dest.orig = *ms.orig
	*ms.orig = ${originName}{}
}`

const messageValueCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct to the dest.
func (ms ${structName}) CopyTo(dest ${structName}) {`

const messageValueCopyToFooterTemplate = `}`

const messageValueTestTemplate = `
func Test${structName}_MoveTo(t *testing.T) {
	ms := generateTest${structName}()
	dest := New${structName}()
	ms.MoveTo(dest)
	assert.EqualValues(t, New${structName}(), ms)
	assert.EqualValues(t, generateTest${structName}(), dest)
}

func Test${structName}_CopyTo(t *testing.T) {
	ms := New${structName}()
	orig := New${structName}()
	orig.CopyTo(ms)
	assert.EqualValues(t, orig, ms)
	orig = generateTest${structName}()
	orig.CopyTo(ms)
	assert.EqualValues(t, orig, ms)
}`

const messageValueGenerateTestTemplate = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	fillTest${structName}(tv)
	return tv
}`

const messageValueFillTestHeaderTemplate = `func fillTest${structName}(tv ${structName}) {`
const messageValueFillTestFooterTemplate = `}`

const messageValueAliasTemplate = `${description}
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} = internal.${structName} 

// New${structName} is an alias for a function to create a new empty ${structName}.
var New${structName} = internal.New${structName}`

const newLine = "\n"

type baseStruct interface {
	getName() string

	generateStruct(sb *strings.Builder)

	generateTests(sb *strings.Builder)

	generateTestValueHelpers(sb *strings.Builder)
}

type aliasGenerator interface {
	generateAlias(sb *strings.Builder)
}

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

func (ms *messageValueStruct) generateAlias(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageValueAliasTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "description":
			return ms.description
		default:
			panic(name)
		}
	}))
	sb.WriteString(newLine + newLine)
}

var _ baseStruct = (*messageValueStruct)(nil)
