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

const accessorSliceTemplate = `// ${fieldName} returns the ${originFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return new${returnType}(&(*ms.orig).${originFieldName})
}`

const accessorsSliceTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	assert.EqualValues(t, New${returnType}(), ms.${fieldName}())
	fillTest${returnType}(ms.${fieldName}())
	testVal${fieldName} := generateTest${returnType}()
	assert.EqualValues(t, testVal${fieldName}, ms.${fieldName}())
}`

const accessorsMessageValueTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return new${returnType}(&(*ms.orig).${originFieldName})
}`

const accessorsMessageValueTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	fillTest${returnType}(ms.${fieldName}())
	assert.EqualValues(t, generateTest${returnType}(), ms.${fieldName}())
}`

const accessorsPrimitiveTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
${extraComment}func (ms ${structName}) ${fieldName}() ${returnType} {
	return (*ms.orig).${originFieldName}
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
${extraComment}func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${originFieldName} = v
}`

const copyToPrimitiveSliceTestTemplate = `	if len(ms.orig.${originFieldName}) == 0 {	
		dest.orig.${originFieldName} = nil
	} else {
		dest.orig.${originFieldName} = make(${rawType}, len(ms.orig.${originFieldName}))
		copy(dest.orig.${originFieldName}, ms.orig.${originFieldName})
	}
`

const accessorsPrimitiveSliceTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ${returnType}{value: (*ms.orig).${originFieldName}}
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${originFieldName} = v.value
}`

const oneOfTypeAccessorHeaderTemplate = `// ${originFieldName}Type returns the type of the ${lowerOriginFieldName} for this ${structName}.
// Calling this function on zero-initialized ${structName} will cause a panic.
func (ms ${structName}) ${originFieldName}Type() ${typeName} {
	switch ms.orig.${originFieldName}.(type) {`

const oneOfTypeAccessorHeaderTestTemplate = `func Test${structName}${originFieldName}Type(t *testing.T) {
	tv := New${structName}()
	assert.Equal(t, ${typeName}None, tv.${originFieldName}Type())
	assert.Equal(t, "", ${typeName}(1000).String())`

const accessorsOneOfMessageTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
//
// Calling this function when ${originOneOfFieldName}Type() != ${typeName} returns an invalid 
// zero-initialized instance of ${returnType}. Note that using such ${returnType} instance can cause panic.
//
// Calling this function on zero-initialized ${structName} will cause a panic.
func (ms ${structName}) ${fieldName}() ${returnType} {
	v, ok := ms.orig.Get${originOneOfFieldName}().(*${originStructType})
	if !ok {
		return ${returnType}{}
	}
	return new${returnType}(v.${originFieldName})
}`

const accessorsOneOfMessageTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	ms.Set${originOneOfFieldName}Type(${typeName})
	fillTest${returnType}(ms.${fieldName}())
	assert.EqualValues(t, generateTest${returnType}(), ms.${fieldName}())
}

func Test${structName}_CopyTo_${fieldName}(t *testing.T) {
	ms := New${structName}()
	ms.Set${originOneOfFieldName}Type(${typeName})
	fillTest${returnType}(ms.${fieldName}())
	dest := New${structName}()
	ms.CopyTo(dest)
	assert.EqualValues(t, ms, dest)
}`

const copyToValueOneOfMessageTemplate = `	case ${typeName}:
		dest.Set${originOneOfFieldName}Type(${typeName})
		ms.${fieldName}().CopyTo(dest.${fieldName}())`

const accessorsOneOfPrimitiveTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return (*ms.orig).Get${originFieldName}()
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${originOneOfFieldName} = &${originStructType}{
		${originFieldName}: v,
	}
}`

const accessorsPrimitiveTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	assert.EqualValues(t, ${defaultVal}, ms.${fieldName}())
	testVal${fieldName} := ${testValue}
	ms.Set${fieldName}(testVal${fieldName})
	assert.EqualValues(t, testVal${fieldName}, ms.${fieldName}())
}`

const accessorsPrimitiveTypedTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ${returnType}((*ms.orig).${originFieldName})
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${originFieldName} = ${rawType}(v)
}`

const accessorsPrimitiveStructTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ${returnType}{orig: ((*ms.orig).${originFieldName})}
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${originFieldName} = v.orig
}`

const accessorsOptionalPrimitiveValueTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return (*ms.orig).Get${fieldName}()
}
// Has${fieldName} returns true if the ${structName} contains a
// ${fieldName} value, false otherwise.
func (ms ${structName}) Has${fieldName}() bool {
	return ms.orig.${fieldName}_ != nil
}
// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	(*ms.orig).${fieldName}_ = &${originStructType}{${fieldName}: v}
}`

type baseField interface {
	generateAccessors(ms baseStruct, sb *strings.Builder)

	generateAccessorsTest(ms baseStruct, sb *strings.Builder)

	generateSetWithTestValue(sb *strings.Builder)

	generateCopyToValue(sb *strings.Builder)
}

type sliceField struct {
	fieldName       string
	originFieldName string
	returnSlice     baseSlice
}

func (sf *sliceField) generateAccessors(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return sf.fieldName
		case "returnType":
			return sf.returnSlice.getName()
		case "originFieldName":
			return sf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsSliceTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return sf.fieldName
		case "returnType":
			return sf.returnSlice.getName()
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\tfillTest" + sf.returnSlice.getName() + "(tv." + sf.fieldName + "())")
}

func (sf *sliceField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tms." + sf.fieldName + "().CopyTo(dest." + sf.fieldName + "())")
}

var _ baseField = (*sliceField)(nil)

type messageValueField struct {
	fieldName       string
	originFieldName string
	returnMessage   *messageValueStruct
}

func (mf *messageValueField) generateAccessors(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsMessageValueTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return mf.fieldName
		case "lowerFieldName":
			return strings.ToLower(mf.fieldName)
		case "returnType":
			return mf.returnMessage.structName
		case "originFieldName":
			return mf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (mf *messageValueField) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsMessageValueTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return mf.fieldName
		case "returnType":
			return mf.returnMessage.structName
		default:
			panic(name)
		}
	}))
}

func (mf *messageValueField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\tfillTest" + mf.returnMessage.structName + "(tv." + mf.fieldName + "())")
}

func (mf *messageValueField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tms." + mf.fieldName + "().CopyTo(dest." + mf.fieldName + "())")
}

var _ baseField = (*messageValueField)(nil)

type primitiveField struct {
	extraComment    string
	fieldName       string
	originFieldName string
	returnType      string
	defaultVal      string
	testVal         string
}

func (pf *primitiveField) generateAccessors(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "extraComment":
			if pf.extraComment != "" {
				return "//\n// " + pf.extraComment + "\n"
			}
			return ""
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

func (pf *primitiveField) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
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

// Types that has defined a custom type (e.g. "type Timestamp uint64")
type primitiveTypedField struct {
	fieldName       string
	originFieldName string
	returnType      string
	defaultVal      string
	testVal         string
	rawType         string
}

func (ptf *primitiveTypedField) generateAccessors(ms baseStruct, sb *strings.Builder) {
	template := accessorsPrimitiveTypedTemplate

	sb.WriteString(os.Expand(template, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
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

func (ptf *primitiveTypedField) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
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

// Types that has defined a custom type (e.g. "type TraceID struct {}")
type primitiveStructField struct {
	fieldName       string
	originFieldName string
	returnType      string
	defaultVal      string
	testVal         string
}

func (ptf *primitiveStructField) generateAccessors(ms baseStruct, sb *strings.Builder) {
	template := accessorsPrimitiveStructTemplate
	sb.WriteString(os.Expand(template, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return ptf.fieldName
		case "lowerFieldName":
			return strings.ToLower(ptf.fieldName)
		case "returnType":
			return ptf.returnType
		case "originFieldName":
			return ptf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (ptf *primitiveStructField) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
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

func (ptf *primitiveStructField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Set" + ptf.fieldName + "(" + ptf.testVal + ")")
}

func (ptf *primitiveStructField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tdest.Set" + ptf.fieldName + "(ms." + ptf.fieldName + "())")
}

var _ baseField = (*primitiveStructField)(nil)

// primitiveSliceField is used to generate fields for slice of primitive types
type primitiveSliceField struct {
	fieldName       string
	originFieldName string
	returnType      string
	defaultVal      string
	rawType         string
	testVal         string
}

func (psf *primitiveSliceField) generateAccessors(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return psf.fieldName
		case "lowerFieldName":
			return strings.ToLower(psf.fieldName)
		case "returnType":
			return psf.returnType
		case "originFieldName":
			return psf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (psf *primitiveSliceField) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "defaultVal":
			return psf.defaultVal
		case "fieldName":
			return psf.fieldName
		case "testValue":
			return psf.testVal
		default:
			panic(name)
		}
	}))
}

func (psf *primitiveSliceField) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\ttv.Set" + psf.fieldName + "(" + psf.testVal + ")")
}

func (psf *primitiveSliceField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString(os.Expand(copyToPrimitiveSliceTestTemplate, func(name string) string {
		switch name {
		case "originFieldName":
			return psf.originFieldName
		case "rawType":
			return psf.rawType
		case "returnType":
			return psf.returnType
		default:
			panic(name)
		}
	}))
}

var _ baseField = (*primitiveStructField)(nil)

type oneOfField struct {
	originTypePrefix string
	originFieldName  string
	typeName         string
	testValueIdx     int
	values           []oneOfValue
}

func (of *oneOfField) generateAccessors(ms baseStruct, sb *strings.Builder) {
	of.generateTypeAccessors(ms, sb)
	sb.WriteString("\n")
	for _, v := range of.values {
		v.generateAccessors(ms, of, sb)
		sb.WriteString("\n")
	}
}

func (of *oneOfField) generateTypeAccessors(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(oneOfTypeAccessorHeaderTemplate, func(name string) string {
		switch name {
		case "lowerOriginFieldName":
			return strings.ToLower(of.originFieldName)
		case "originFieldName":
			return of.originFieldName
		case "structName":
			return ms.getName()
		case "typeName":
			return of.typeName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
	for _, v := range of.values {
		v.generateTypeSwitchCase(of, sb)
	}
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn " + of.typeName + "None\n")
	sb.WriteString("}\n")
}

func (of *oneOfField) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	of.generateTypeAccessorsTest(ms, sb)
	sb.WriteString("\n")
	for _, v := range of.values {
		v.generateTests(ms.(*messageValueStruct), of, sb)
		sb.WriteString("\n")
	}
}

func (of *oneOfField) generateTypeAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(oneOfTypeAccessorHeaderTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "originFieldName":
			return of.originFieldName
		case "typeName":
			return of.typeName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
	for _, v := range of.values {
		if mv, ok := v.(*oneOfMessageValue); ok {
			sb.WriteString("\tassert.Equal(t, " + mv.fieldName + "{}, tv." + mv.fieldName + "())\n")
		}
	}
	for _, v := range of.values {
		v.generateSetWithTestValue(of, sb)
		sb.WriteString("\n\tassert.Equal(t, " + of.typeName + v.getFieldType() + ", " +
			"tv." + of.originFieldName + "Type())\n")
	}
	sb.WriteString("}\n")
}

func (of *oneOfField) generateSetWithTestValue(sb *strings.Builder) {
	of.values[of.testValueIdx].generateSetWithTestValue(of, sb)
}

func (of *oneOfField) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("\tswitch ms." + of.originFieldName + "Type() {\n")
	for _, v := range of.values {
		v.generateCopyToValue(of, sb)
	}
	sb.WriteString("\t}\n")
}

var _ baseField = (*oneOfField)(nil)

type oneOfValue interface {
	getFieldType() string
	generateAccessors(ms baseStruct, of *oneOfField, sb *strings.Builder)
	generateTests(ms baseStruct, of *oneOfField, sb *strings.Builder)
	generateSetWithTestValue(of *oneOfField, sb *strings.Builder)
	generateCopyToValue(of *oneOfField, sb *strings.Builder)
	generateTypeSwitchCase(of *oneOfField, sb *strings.Builder)
}

type oneOfPrimitiveValue struct {
	fieldName       string
	fieldType       string
	defaultVal      string
	testVal         string
	returnType      string
	originFieldName string
}

func (opv *oneOfPrimitiveValue) getFieldType() string {
	return opv.fieldType
}

func (opv *oneOfPrimitiveValue) generateAccessors(ms baseStruct, of *oneOfField, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsOneOfPrimitiveTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return opv.fieldName
		case "lowerFieldName":
			return strings.ToLower(opv.fieldName)
		case "returnType":
			return opv.returnType
		case "originFieldName":
			return opv.originFieldName
		case "originOneOfFieldName":
			return of.originFieldName
		case "originStructType":
			return of.originTypePrefix + opv.originFieldName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (opv *oneOfPrimitiveValue) generateTests(ms baseStruct, _ *oneOfField, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "defaultVal":
			return opv.defaultVal
		case "fieldName":
			return opv.fieldName
		case "testValue":
			return opv.testVal
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (opv *oneOfPrimitiveValue) generateSetWithTestValue(_ *oneOfField, sb *strings.Builder) {
	sb.WriteString("\t tv.Set" + opv.fieldName + "(" + opv.testVal + ")")
}

func (opv *oneOfPrimitiveValue) generateCopyToValue(of *oneOfField, sb *strings.Builder) {
	sb.WriteString("\tcase " + of.typeName + opv.fieldType + ":\n")
	sb.WriteString("\t dest.Set" + opv.fieldName + "(ms." + opv.fieldName + "())\n")
}

func (opv *oneOfPrimitiveValue) generateTypeSwitchCase(of *oneOfField, sb *strings.Builder) {
	sb.WriteString("\tcase *" + of.originTypePrefix + opv.originFieldName + ":\n")
	sb.WriteString("\t\treturn " + of.typeName + opv.fieldType + "\n")
}

var _ oneOfValue = (*oneOfPrimitiveValue)(nil)

type oneOfMessageValue struct {
	fieldName       string
	originFieldName string
	returnMessage   *messageValueStruct
}

func (omv *oneOfMessageValue) getFieldType() string {
	return omv.fieldName
}

func (omv *oneOfMessageValue) generateAccessors(ms baseStruct, of *oneOfField, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsOneOfMessageTemplate, func(name string) string {
		switch name {
		case "fieldName":
			return omv.fieldName
		case "lowerFieldName":
			return strings.ToLower(omv.fieldName)
		case "originFieldName":
			return omv.originFieldName
		case "originOneOfFieldName":
			return of.originFieldName
		case "originStructType":
			return of.originTypePrefix + omv.originFieldName
		case "returnType":
			return omv.returnMessage.structName
		case "structName":
			return ms.getName()
		case "typeName":
			return of.typeName + omv.returnMessage.structName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (omv *oneOfMessageValue) generateTests(ms baseStruct, of *oneOfField, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsOneOfMessageTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return omv.fieldName
		case "returnType":
			return omv.returnMessage.structName
		case "originOneOfFieldName":
			return of.originFieldName
		case "typeName":
			return of.typeName + omv.returnMessage.structName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (omv *oneOfMessageValue) generateSetWithTestValue(of *oneOfField, sb *strings.Builder) {
	sb.WriteString("tv.Set" + of.originFieldName + "Type(" + of.typeName + omv.returnMessage.structName + ")\n")
	sb.WriteString("fillTest" + omv.returnMessage.structName + "(tv." + omv.fieldName + "())")
}

func (omv *oneOfMessageValue) generateCopyToValue(of *oneOfField, sb *strings.Builder) {
	sb.WriteString(os.Expand(copyToValueOneOfMessageTemplate, func(name string) string {
		switch name {
		case "fieldName":
			return omv.fieldName
		case "originOneOfFieldName":
			return of.originFieldName
		case "typeName":
			return of.typeName + omv.fieldName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (omv *oneOfMessageValue) generateTypeSwitchCase(of *oneOfField, sb *strings.Builder) {
	sb.WriteString("\tcase *" + of.originTypePrefix + omv.originFieldName + ":\n")
	sb.WriteString("\t\treturn " + of.typeName + omv.fieldName + "\n")
}

var _ oneOfValue = (*oneOfMessageValue)(nil)

type optionalPrimitiveValue struct {
	fieldName        string
	fieldType        string
	defaultVal       string
	testVal          string
	returnType       string
	originFieldName  string
	originTypePrefix string
}

func (opv *optionalPrimitiveValue) generateAccessors(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsOptionalPrimitiveValueTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return opv.fieldName
		case "lowerFieldName":
			return strings.ToLower(opv.fieldName)
		case "returnType":
			return opv.returnType
		case "originFieldName":
			return opv.originFieldName
		case "originStructType":
			return opv.originTypePrefix + opv.originFieldName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (opv *optionalPrimitiveValue) generateAccessorsTest(ms baseStruct, sb *strings.Builder) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "defaultVal":
			return opv.defaultVal
		case "fieldName":
			return opv.fieldName
		case "testValue":
			return opv.testVal
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (opv *optionalPrimitiveValue) generateSetWithTestValue(sb *strings.Builder) {
	sb.WriteString("\t tv.Set" + opv.fieldName + "(" + opv.testVal + ")")
}

func (opv *optionalPrimitiveValue) generateCopyToValue(sb *strings.Builder) {
	sb.WriteString("if ms.Has" + opv.fieldName + "(){\n")
	sb.WriteString("\tdest.Set" + opv.fieldName + "(ms." + opv.fieldName + "())\n")
	sb.WriteString("}\n")
}

var _ baseField = (*optionalPrimitiveValue)(nil)
