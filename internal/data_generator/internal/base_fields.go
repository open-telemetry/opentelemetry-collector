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
func (ms ${structName}) ${fieldName}() ${returnType} {
	return new${returnType}(&ms.orig.${originFieldName})
}

// Set${fieldName} replaces the ${originFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.orig.${originFieldName} = *v.orig
}`

const accessorMessageTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	if rm.orig.${originFieldName} == nil {
		// No ${originFieldName} available, initialize one to make all operations on ${returnType} available.
		rm.orig.${originFieldName} = &${structOriginFullName}{}
	}
	return new${returnType}(&ms.orig.${originFieldName})
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.orig.${originFieldName} = *v.orig
}`

const accessorPrimitiveTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ms.orig.Get${originFieldName}()
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.orig.${originFieldName} = v
}`

const accessorPrimitiveTypedTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ${returnType}(ms.orig.Get${originFieldName}())
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.orig.${originFieldName} = ${rawType}(v)
}`

type baseField interface {
	generateAccessors(ms *messageStruct, sb *strings.Builder)
}

type sliceField struct {
	fieldMame       string
	originFieldName string
	returnSlice     *sliceStruct
}

func (mf *sliceField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "fieldName":
			return mf.fieldMame
		case "returnType":
			return mf.returnSlice.structName
		case "originFieldName":
			return mf.originFieldName
		default:
			panic(name)
		}
	}))
}

var _ baseField = (*sliceField)(nil)

type messageField struct {
	fieldMame       string
	originFieldName string
	returnMessage   *messageStruct
}

func (mf *messageField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorMessageTemplate, func(name string) string {
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

var _ baseField = (*messageField)(nil)

type primitiveField struct {
	fieldMame       string
	originFieldName string
	returnType      string
}

func (pf *primitiveField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorPrimitiveTemplate, func(name string) string {
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

var _ baseField = (*primitiveField)(nil)

// Types that has defined a custom type (e.g. "type TimestampUnixNano uint64")
type primitiveTypedField struct {
	fieldMame       string
	originFieldName string
	returnType      string
	rawType         string
}

func (ptf *primitiveTypedField) generateAccessors(ms *messageStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorPrimitiveTypedTemplate, func(name string) string {
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

var _ baseField = (*primitiveTypedField)(nil)
