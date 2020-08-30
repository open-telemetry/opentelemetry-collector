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

const accessorSliceTemplate = `// ${fieldName} returns the ${originFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return new${returnType}(&(*ms.orig).${originFieldName})
}`

const accessorsSliceTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	ms.InitEmpty()
	assert.EqualValues(t, New${returnType}(), ms.${fieldName}())
	fillTest${returnType}(ms.${fieldName}())
	testVal${fieldName} := generateTest${returnType}()
	assert.EqualValues(t, testVal${fieldName}, ms.${fieldName}())
}`

const accessorsMessageTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
// If no ${lowerFieldName} available, it creates an empty message and associates it with this ${structName}.
//
//  Empty initialized ${structName} will return "nil" ${returnType}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return new${returnType}(&(*ms.orig).${originFieldName})
}`

const accessorsMessageTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	ms.InitEmpty()
	assert.EqualValues(t, true, ms.${fieldName}().IsNil())
	ms.${fieldName}().InitEmpty()
	assert.EqualValues(t, false, ms.${fieldName}().IsNil())
	fillTest${returnType}(ms.${fieldName}())
	assert.EqualValues(t, generateTest${returnType}(), ms.${fieldName}())
}`

const accessorsPrimitiveTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return (*ms.orig).${originFieldName}
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${originFieldName} = v
}`

const accessorsPrimitiveTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	ms.InitEmpty()
	assert.EqualValues(t, ${defaultVal}, ms.${fieldName}())
	testVal${fieldName} := ${testValue}
	ms.Set${fieldName}(testVal${fieldName})
	assert.EqualValues(t, testVal${fieldName}, ms.${fieldName}())
}`

const accessorsPrimitiveTypedTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ${returnType}((*ms.orig).${originFieldName})
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${originFieldName} = ${rawType}(v)
}`

type baseField interface {
	generateAccessors(ms *messageStruct, sb *strings.Builder)

	generateAccessorsTest(ms *messageStruct, sb *strings.Builder)

	generateSetWithTestValue(sb *strings.Builder)

	generateCopyToValue(sb *strings.Builder)
}

type sliceField struct {
	fieldName       string
	originFieldName string
	returnSlice     *sliceStruct
}

func (sf *sliceField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return sf.fieldName
		case "returnType":
			return sf.returnSlice.structName
		case "originFieldName":
			return sf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateAccessorsTest(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsSliceTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return sf.fieldName
		case "returnType":
			return sf.returnSlice.structName
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\tfillTest" + sf.returnSlice.structName + "(tv." + sf.fieldName + "())")
}

func (sf *sliceField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tms." + sf.fieldName + "().CopyTo(dest." + sf.fieldName + "())")
}

var _ baseField = (*sliceField)(nil)

type messageField struct {
	fieldName       string
	originFieldName string
	returnMessage   *messageStruct
}

func (mf *messageField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsMessageTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return mf.fieldName
		case "lowerFieldName":
			return strings.ToLower(mf.fieldName)
		case "returnType":
			return mf.returnMessage.structName
		case "structOriginFullName":
			return mf.returnMessage.originFullName
		case "originFieldName":
			return mf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (mf *messageField) generateAccessorsTest(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsMessageTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return mf.fieldName
		case "returnType":
			return mf.returnMessage.structName
		default:
			panic(name)
		}
	}))
}

func (mf *messageField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv." + mf.fieldName + "().InitEmpty()\n")
	sb.WriteString("\tfillTest" + mf.returnMessage.structName + "(tv." + mf.fieldName + "())")
}

func (mf *messageField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tms." + mf.fieldName + "().CopyTo(dest." + mf.fieldName + "())")
}

var _ baseField = (*messageField)(nil)

type primitiveField struct {
	fieldName       string
	originFieldName string
	returnType      string
	defaultVal      string
	testVal         string
}

func (pf *primitiveField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return pf.fieldName
		case "lowerFieldName":
			return strings.ToLower(pf.fieldName)
		case "returnType":
			return pf.returnType
		case "originFieldName":
			return pf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (pf *primitiveField) generateAccessorsTest(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "defaultVal":
			return pf.defaultVal
		case "fieldName":
			return pf.fieldName
		case "testValue":
			return pf.testVal
		default:
			panic(name)
		}
	}))
}

func (pf *primitiveField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Set" + pf.fieldName + "(" + pf.testVal + ")")
}

func (pf *primitiveField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tdest.Set" + pf.fieldName + "(ms." + pf.fieldName + "())")
}

var _ baseField = (*primitiveField)(nil)

// Types that has defined a custom type (e.g. "type TimestampUnixNano uint64")
type primitiveTypedField struct {
	fieldName       string
	originFieldName string
	returnType      string
	defaultVal      string
	testVal         string
	rawType         string
}

func (ptf *primitiveTypedField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTypedTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return ptf.fieldName
		case "lowerFieldName":
			return strings.ToLower(ptf.fieldName)
		case "returnType":
			return ptf.returnType
		case "rawType":
			return ptf.rawType
		case "originFieldName":
			return ptf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (ptf *primitiveTypedField) generateAccessorsTest(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "defaultVal":
			return ptf.defaultVal
		case "fieldName":
			return ptf.fieldName
		case "testValue":
			return ptf.testVal
		default:
			panic(name)
		}
	}))
}

func (ptf *primitiveTypedField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Set" + ptf.fieldName + "(" + ptf.testVal + ")")
}

func (ptf *primitiveTypedField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tdest.Set" + ptf.fieldName + "(ms." + ptf.fieldName + "())")
}

var _ baseField = (*primitiveTypedField)(nil)
