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
	orig *${originName}
}

func ${newPrefix}${structName}FromOrig(orig *${originName}) ${structName} {
	return ${structName}{common${structName}{orig}}
}

func ${newPrefix}Mutable${structName}FromOrig(orig *${originName}) Mutable${structName} {
	return Mutable${structName}{common${structName}: common${structName}{orig}}
}

// NewMutable${structName} creates a new empty ${structName}.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewMutable${structName}() Mutable${structName} {
	return ${newPrefix}Mutable${structName}FromOrig(&${originName}{})
}

// nolint:unused
func (ms ${structName}) asMutable() Mutable${structName} {
	return Mutable${structName}{common${structName}: common${structName}{orig: ms.orig}}
}

func (ms Mutable${structName}) AsImmutable() ${structName} {
	return ${structName}{common${structName}{orig: ms.orig}}
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms Mutable${structName}) MoveTo(dest Mutable${structName}) {
	*dest.orig = *ms.orig
	*ms.orig = ${originName}{}
}`

const messageValueCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct overriding the destination.
func (ms common${structName}) CopyTo(dest Mutable${structName}) {`

const messageValueCopyToFooterTemplate = `}`

const messageValueTestTemplate = `
func Test${structName}_MoveTo(t *testing.T) {
	ms := ${generateTestPrefix}${structName}()
	dest := NewMutable${structName}()
	ms.MoveTo(dest)
	assert.Equal(t, NewMutable${structName}(), ms)
	assert.Equal(t, ${generateTestPrefix}${structName}(), dest)
}

func Test${structName}_CopyTo(t *testing.T) {
	ms := NewMutable${structName}()
	orig := NewMutable${structName}()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = ${generateTestPrefix}${structName}()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}`

const messageValueGenerateTestTemplate = `func ${generateTestPrefix}${structName}() Mutable${structName} {
	tv := NewMutable${structName}()
	${fillTestPrefix}${structName}(tv)
	return tv
}`

const aliasTemplate = `
type ${structName} = internal.${structName}

type Mutable${structName} = internal.Mutable${structName}

var NewMutable${structName} = internal.NewMutable${structName}`

const newLine = "\n"

type baseStruct interface {
	getName() string
	getPackageName() string
	generateStruct(sb *bytes.Buffer)
	generateTests(sb *bytes.Buffer)
	generateTestValueHelpers(sb *bytes.Buffer)
	generateAliases(sb *bytes.Buffer)
}

// messageValueStruct generates a struct for a proto message. The struct can be generated both as a common struct
// that can be used as a field in struct from other packages and as an isolated struct with depending on a package name.
type messageValueStruct struct {
	structName     string
	packageName    string
	description    string
	originFullName string
	fields         []baseField
}

func (ms *messageValueStruct) getName() string {
	return ms.structName
}

func (ms *messageValueStruct) getPackageName() string {
	return ms.packageName
}

func (ms *messageValueStruct) generateStruct(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueTemplate, ms.templateFields()))

	// Write accessors for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine + newLine)
		f.generateAccessors(ms, sb)
	}
	sb.WriteString(newLine + newLine)
	sb.WriteString(os.Expand(messageValueCopyToHeaderTemplate, ms.templateFields()))

	// Write accessors CopyTo for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine)
		f.generateCopyToValue(ms, sb)
	}
	sb.WriteString(newLine)
	sb.WriteString(messageValueCopyToFooterTemplate)
}

func (ms *messageValueStruct) generateTests(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueTestTemplate, ms.templateFields()))

	// Write accessors tests for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine + newLine)
		f.generateAccessorsTest(ms, sb)
	}
}

func (ms *messageValueStruct) generateTestValueHelpers(sb *bytes.Buffer) {
	// Write generateTest func for the struct
	sb.WriteString(os.Expand(messageValueGenerateTestTemplate, ms.templateFields()))

	// Write fillTest func for the struct
	sb.WriteString(newLine + newLine + "func ")
	if usedByOtherDataTypes(ms.packageName) {
		sb.WriteString("FillTest")
	} else {
		sb.WriteString("fillTest")
	}
	sb.WriteString(ms.structName + "(tv Mutable" + ms.structName + ") {")
	for _, f := range ms.fields {
		sb.WriteString(newLine)
		f.generateSetWithTestValue(ms, sb)
	}
	sb.WriteString(newLine + "}" + newLine)
}

func (ms *messageValueStruct) generateAliases(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(aliasTemplate, ms.templateFields()))
	sb.WriteString(newLine + newLine)
}

func (ms *messageValueStruct) templateFields() func(name string) string {
	return func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		case "generateTestPrefix":
			return generateTestPrefix(ms.packageName)
		case "fillTestPrefix":
			return fillTestPrefix(ms.packageName)
		case "description":
			return ms.description
		case "newPrefix":
			return newPrefix(ms.packageName)
		default:
			panic(name)
		}
	}
}

var _ baseStruct = (*messageValueStruct)(nil)

func newPrefix(packageName string) string {
	if usedByOtherDataTypes(packageName) {
		return "New"
	}
	return "new"
}

func fillTestPrefix(packageName string) string {
	if usedByOtherDataTypes(packageName) {
		return "FillTest"
	}
	return "fillTest"
}

func generateTestPrefix(packageName string) string {
	if usedByOtherDataTypes(packageName) {
		return "GenerateTest"
	}
	return "generateTest"

}
