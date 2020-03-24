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

const accessorSliceTemplate = `// ${fieldName} returns the ${originFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return new${returnType}(&ms.orig.${originFieldName})
}

// Set${fieldName} replaces the ${originFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.orig.${originFieldName} = *v.orig
}`

const accessorsSliceTestTemplate = `	assert.EqualValues(t, New${returnType}(${constructorDefaultValue}), ms.${fieldName}())
	testVal${fieldName} := generateTest${returnType}()
	ms.Set${fieldName}(testVal${fieldName})
	assert.EqualValues(t, testVal${fieldName}, ms.${fieldName}())`

const accessorsMessageTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
// If no ${lowerFieldName} available, it creates an empty message and associates it with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return new${returnType}(ms.orig.${originFieldName})
}

// Init${fieldName}IfNil() initialize the ${lowerFieldName} with an empty message if and only if
// the current value is "nil".
func (ms ${structName}) Init${fieldName}IfNil() {
	if ms.orig.${originFieldName} == nil {
		ms.orig.${originFieldName} = &${structOriginFullName}{}
	}
}`

const accessorsMessageTestTemplate = `	assert.EqualValues(t, true, ms.${fieldName}().IsNil())
	ms.Init${fieldName}IfNil()
	assert.EqualValues(t, false, ms.${fieldName}().IsNil())
	assert.EqualValues(t, NewEmpty${returnType}(), ms.${fieldName}())
	testVal${fieldName} := generateTest${returnType}()
	fillTest${returnType}(ms.${fieldName}())
	assert.EqualValues(t, testVal${fieldName}, ms.${fieldName}())
	assert.EqualValues(t, false, ms.${fieldName}().IsNil())`

const accessorsPrimitiveTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ms.orig.${originFieldName}
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.orig.${originFieldName} = v
}`

const accessorsPrimitiveTestTemplate = `	assert.EqualValues(t, ${defaultVal}, ms.${fieldName}())
	testVal${fieldName} := ${testValue}
	ms.Set${fieldName}(testVal${fieldName})
	assert.EqualValues(t, testVal${fieldName}, ms.${fieldName}())`

const accessorsPrimitiveTypedTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ${returnType}(ms.orig.${originFieldName})
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
//
// Important: This causes a runtime error if IsNil() returns "true".
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.orig.${originFieldName} = ${rawType}(v)
}`

type baseField interface {
	generateAccessors(ms *messageStruct, sb *strings.Builder)

	generateAccessorsTests(ms *messageStruct, sb *strings.Builder)

	generateSetWithTestValue(sb *strings.Builder)
}

type sliceField struct {
	fieldMame               string
	originFieldName         string
	returnSlice             *sliceStruct
	constructorDefaultValue string
}

func (sf *sliceField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return sf.fieldMame
		case "returnType":
			return sf.returnSlice.structName
		case "originFieldName":
			return sf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateAccessorsTests(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsSliceTestTemplate, func(name string) string {
		switch name {
		case "fieldName":
			return sf.fieldMame
		case "returnType":
			return sf.returnSlice.structName
		case "constructorDefaultValue":
			return sf.constructorDefaultValue
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Set" + sf.fieldMame + "(generateTest" + sf.returnSlice.structName + "())")
}

var _ baseField = (*sliceField)(nil)

type messageField struct {
	fieldMame       string
	originFieldName string
	returnMessage   *messageStruct
}

func (mf *messageField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsMessageTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return mf.fieldMame
		case "lowerFieldName":
			return strings.ToLower(mf.fieldMame)
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

func (mf *messageField) generateAccessorsTests(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsMessageTestTemplate, func(name string) string {
		switch name {
		case "fieldName":
			return mf.fieldMame
		case "returnType":
			return mf.returnMessage.structName
		default:
			panic(name)
		}
	}))
}

func (mf *messageField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Init" + mf.fieldMame + "IfNil()\n")
	sb.WriteString("\tfillTest" + mf.returnMessage.structName + "(tv." + mf.fieldMame + "())")
}

var _ baseField = (*messageField)(nil)

type primitiveField struct {
	fieldMame       string
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
			return pf.fieldMame
		case "lowerFieldName":
			return strings.ToLower(pf.fieldMame)
		case "returnType":
			return pf.returnType
		case "originFieldName":
			return pf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (pf *primitiveField) generateAccessorsTests(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "defaultVal":
			return pf.defaultVal
		case "fieldName":
			return pf.fieldMame
		case "testValue":
			return pf.testVal
		default:
			panic(name)
		}
	}))
}

func (pf *primitiveField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Set" + pf.fieldMame + "(" + pf.testVal + ")")
}

var _ baseField = (*primitiveField)(nil)

// Types that has defined a custom type (e.g. "type TimestampUnixNano uint64")
type primitiveTypedField struct {
	fieldMame       string
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
			return ptf.fieldMame
		case "lowerFieldName":
			return strings.ToLower(ptf.fieldMame)
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

func (ptf *primitiveTypedField) generateAccessorsTests(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "defaultVal":
			return ptf.defaultVal
		case "fieldName":
			return ptf.fieldMame
		case "testValue":
			return ptf.testVal
		default:
			panic(name)
		}
	}))
}

func (ptf *primitiveTypedField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Set" + ptf.fieldMame + "(" + ptf.testVal + ")")
}

var _ baseField = (*primitiveTypedField)(nil)
