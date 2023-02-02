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
type ${structName} interface {
	common${structName}
${notMutatingMethods}}

type Mutable${structName} interface {
	common${structName}
${mutatingMethods}}

type common${structName} interface {
	getOrig() *${originName}
${commonMethods}}

type immutable${structName} struct {
	orig *${originName}
}

type mutable${structName} struct {
	immutable${structName}
}

func ${newPrefix}Immutable${structName}(orig *${originName}) immutable${structName} {
	return immutable${structName}{orig}
}

func ${newPrefix}Mutable${structName}(orig *${originName}) mutable${structName} {
	return mutable${structName}{immutable${structName}{orig}}
}

func (ms immutable${structName}) getOrig() *${originName} {
	return ms.orig
}

// New${structName} creates a new empty ${structName}.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func New${structName}() Mutable${structName} {
	return ${newPrefix}Mutable${structName}(&${originName}{})
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms mutable${structName}) MoveTo(dest Mutable${structName}) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = ${originName}{}
}`

const messageValueCopyToHeaderTemplate = `// CopyTo copies all properties from the current struct overriding the destination.
func (ms immutable${structName}) CopyTo(dest Mutable${structName}) {`

const messageValueCopyToFooterTemplate = `}`

const messageValueTestTemplate = `
func Test${structName}_MoveTo(t *testing.T) {
	ms := ${generateTestPrefix}${structName}()
	dest := New${structName}()
	ms.MoveTo(dest)
	assert.Equal(t, New${structName}(), ms)
	assert.Equal(t, ${generateTestPrefix}${structName}(), dest)
}

func Test${structName}_CopyTo(t *testing.T) {
	ms := New${structName}()
	orig := New${structName}()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = ${generateTestPrefix}${structName}()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
}`

const messageValueGenerateTestTemplate = `func ${generateTestPrefix}${structName}() Mutable${structName} {
	tv := New${structName}()
	${fillTestPrefix}${structName}(tv)
	return tv
}`

const aliasTemplate = `
type ${structName} = internal.${structName}

type Mutable${structName} = internal.Mutable${structName}

var New${structName} = internal.New${structName}`

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
	sb.WriteString(os.Expand(messageValueTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		case "description":
			return ms.description
		case "commonMethods":
			return ms.commonMethods()
		case "notMutatingMethods":
			return ms.notMutatingMethods()
		case "mutatingMethods":
			return ms.mutatingMethods()
		case "newPrefix":
			return newPrefix(ms.packageName)
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

func (ms *messageValueStruct) commonMethods() string {
	var sb bytes.Buffer
	sb.WriteString("\tCopyTo(dest Mutable" + ms.structName + ")" + newLine)
	for _, f := range ms.fields {
		f.generateCommonSignatures(ms, &sb)
	}
	return sb.String()
}

func (ms *messageValueStruct) notMutatingMethods() string {
	var sb bytes.Buffer
	for _, f := range ms.fields {
		f.generateNotMutatingSignatures(ms, &sb)
	}
	return sb.String()
}

func (ms *messageValueStruct) mutatingMethods() string {
	var sb bytes.Buffer
	sb.WriteString("\tMoveTo(dest Mutable" + ms.structName + ")" + newLine)
	for _, f := range ms.fields {
		f.generateMutatingSignatures(ms, &sb)
	}
	return sb.String()
}

func (ms *messageValueStruct) generateTests(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(messageValueTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "generateTestPrefix":
			return generateTestPrefix(ms.packageName)
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
	// Write generateTest func for the struct
	sb.WriteString(os.Expand(messageValueGenerateTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		case "generateTestPrefix":
			return generateTestPrefix(ms.packageName)
		case "fillTestPrefix":
			return fillTestPrefix(ms.packageName)
		default:
			panic(name)
		}
	}))

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
	sb.WriteString(os.Expand(aliasTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		default:
			panic(name)
		}
	}))
	sb.WriteString(newLine + newLine)
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
