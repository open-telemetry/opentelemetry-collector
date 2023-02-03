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
${typeDeclaration}

func new${structName}(orig *${originName}) ${structName} {
	return ${newFuncValue}
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
	*dest.${origAccessor} = *ms.${origAccessor}
	*ms.${origAccessor} = ${originName}{}
}`

const messageValueGetOrigTemplate = `func (ms ${structName}) getOrig() *${originName} {
	return internal.GetOrig${structName}(internal.${structName}(ms))
}`

const messageValueCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct overriding the destination.
func (ms ${structName}) CopyTo(dest ${structName}) {`

const messageValueCopyToFooterTemplate = `}`

const messageValueTestTemplate = `
func Test${structName}_MoveTo(t *testing.T) {
	ms := ${generateTestData}
	dest := New${structName}()
	ms.MoveTo(dest)
	assert.Equal(t, New${structName}(), ms)
	assert.Equal(t, ${generateTestData}, dest)
}

func Test${structName}_CopyTo(t *testing.T) {
	ms := New${structName}()
	orig := New${structName}()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = ${generateTestData}
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}`

const messageValueGenerateTestTemplate = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	fillTest${structName}(tv)
	return tv
}`

const messageValueGenerateTestTemplateCommon = `func GenerateTest${structName}() ${structName} {
	orig := ${originName}{}
	tv := New${structName}(&orig)
	FillTest${structName}(tv)
	return tv
}`

const messageValueAliasTemplate = `
type ${structName} struct {
	orig *${originName}
}

func GetOrig${structName}(ms ${structName}) *${originName} {
	return ms.orig
}

func New${structName}(orig *${originName}) ${structName} {
	return ${structName}{orig: orig}
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
	if usedByOtherDataTypes(ms.packageName) {
		sb.WriteString(newLine + newLine)
		sb.WriteString(os.Expand(messageValueGetOrigTemplate, ms.templateFields()))
	}

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
	template := messageValueGenerateTestTemplate
	if usedByOtherDataTypes(ms.packageName) {
		template = messageValueGenerateTestTemplateCommon
	}
	sb.WriteString(os.Expand(template, ms.templateFields()))

	// Write fillTest func for the struct
	sb.WriteString(newLine + newLine + "func ")
	if usedByOtherDataTypes(ms.packageName) {
		sb.WriteString("FillTest")
	} else {
		sb.WriteString("fillTest")
	}
	sb.WriteString(ms.structName + "(tv " + ms.structName + ") {")
	for _, f := range ms.fields {
		sb.WriteString(newLine)
		f.generateSetWithTestValue(ms, sb)
	}
	sb.WriteString(newLine + "}" + newLine)
}

func (ms *messageValueStruct) generateInternal(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueAliasTemplate, ms.templateFields()))
	sb.WriteString(newLine + newLine)
}

func (ms *messageValueStruct) templateFields() func(name string) string {
	return func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		case "generateTestData":
			if usedByOtherDataTypes(ms.packageName) {
				return ms.structName + "(internal.GenerateTest" + ms.structName + "())"
			}
			return "generateTest" + ms.structName + "()"
		case "description":
			return ms.description
		case "typeDeclaration":
			if usedByOtherDataTypes(ms.packageName) {
				return "type " + ms.structName + " internal." + ms.structName
			}
			return "type " + ms.structName + " struct {\n\torig *" + ms.originFullName + "\n}"
		case "newFuncValue":
			if usedByOtherDataTypes(ms.packageName) {
				return ms.structName + "(internal.New" + ms.structName + "(orig))"
			}
			return ms.structName + "{orig}"
		case "origAccessor":
			return origAccessor(ms)
		default:
			panic(name)
		}
	}
}

var _ baseStruct = (*messageValueStruct)(nil)
