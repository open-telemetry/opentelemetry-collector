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
	"strings"
)

const accessorSliceTemplate = `// ${fieldName} returns the ${fieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${packageName}${returnType} {
	return ${packageName}${returnType}(internal.New${returnType}(&ms.getOrig().${fieldName}))
}`

const accessorsSliceTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	assert.Equal(t, ${packageName}New${returnType}(), ms.${fieldName}())
	internal.FillTest${returnType}(internal.${returnType}(ms.${fieldName}()))
	assert.Equal(t, ${packageName}${returnType}(internal.GenerateTest${returnType}()), ms.${fieldName}())
}`

const accessorsMessageValueTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${packageName}${returnType} {
	return ${packageName}${returnType}(internal.New${returnType}(&ms.getOrig().${fieldName}))
}`

const accessorsMessageValueTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	internal.FillTest${returnType}(internal.${returnType}(ms.${fieldName}()))
	assert.Equal(t, ${packageName}${returnType}(internal.GenerateTest${returnType}()), ms.${fieldName}())
}`

const accessorsPrimitiveTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${packageName}${returnType} {
	return ms.getOrig().${fieldName}
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.getOrig().${fieldName} = v
}`

const accessorsPrimitiveSliceTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${packageName}${returnType} {
	return ${packageName}${returnType}(internal.New${returnType}(&ms.getOrig().${fieldName}))
}`

const oneOfTypeAccessorHeaderTemplate = `// ${typeFuncName} returns the type of the ${lowerOriginFieldName} for this ${structName}.
// Calling this function on zero-initialized ${structName} will cause a panic.
func (ms ${structName}) ${typeFuncName}() ${typeName} {
	switch ms.getOrig().${originFieldName}.(type) {`

const oneOfTypeAccessorHeaderTestTemplate = `func Test${structName}_${typeFuncName}(t *testing.T) {
	tv := New${structName}()
	assert.Equal(t, ${typeName}Empty, tv.${typeFuncName}())
}
`

const accessorsOneOfMessageTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
//
// Calling this function when ${originOneOfTypeFuncName}() != ${typeName} returns an invalid 
// zero-initialized instance of ${returnType}. Note that using such ${returnType} instance can cause panic.
//
// Calling this function on zero-initialized ${structName} will cause a panic.
func (ms ${structName}) ${fieldName}() ${returnType} {
	v, ok := ms.getOrig().Get${originOneOfFieldName}().(*${originStructType})
	if !ok {
		return ${returnType}{}
	}
	return new${returnType}(v.${fieldName})
}

// SetEmpty${fieldName} sets an empty ${lowerFieldName} to this ${structName}.
//
// After this, ${originOneOfTypeFuncName}() function will return ${typeName}".
//
// Calling this function on zero-initialized ${structName} will cause a panic.
func (ms ${structName}) SetEmpty${fieldName}() ${returnType} {
	val := &${originFieldPackageName}.${fieldName}{}
	ms.getOrig().${originOneOfFieldName} = &${originStructType}{${fieldName}: val}
	return new${returnType}(val)
}`

const accessorsOneOfMessageTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	internal.FillTest${returnType}(internal.${returnType}(ms.SetEmpty${fieldName}()))
	assert.Equal(t, ${typeName}, ms.${originOneOfTypeFuncName}())
	assert.Equal(t, ${returnType}(internal.GenerateTest${returnType}()), ms.${fieldName}())
}

func Test${structName}_CopyTo_${fieldName}(t *testing.T) {
	ms := New${structName}()
	internal.FillTest${returnType}(internal.${returnType}(ms.SetEmpty${fieldName}()))
	dest := New${structName}()
	ms.CopyTo(dest)
	assert.Equal(t, ms, dest)
}`

const copyToValueOneOfMessageTemplate = `	case ${typeName}:
		ms.${fieldName}().CopyTo(dest.SetEmpty${fieldName}())`

const accessorsOneOfPrimitiveTemplate = `// ${accessorFieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${accessorFieldName}() ${returnType} {
	return ms.getOrig().Get${originFieldName}()
}

// Set${accessorFieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${accessorFieldName}(v ${returnType}) {
	ms.getOrig().${originOneOfFieldName} = &${originStructType}{
		${originFieldName}: v,
	}
}`

const accessorsOneOfPrimitiveTestTemplate = `func Test${structName}_${accessorFieldName}(t *testing.T) {
	ms := New${structName}()
	assert.Equal(t, ${defaultVal}, ms.${accessorFieldName}())
	ms.Set${accessorFieldName}(${testValue})
	assert.Equal(t, ${testValue}, ms.${accessorFieldName}())
	assert.Equal(t, ${typeName}, ms.${originOneOfTypeFuncName}())
}`

const accessorsPrimitiveTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	assert.Equal(t, ${defaultVal}, ms.${fieldName}())
	ms.Set${fieldName}(${testValue})
	assert.Equal(t, ${testValue}, ms.${fieldName}())
}`

const accessorsPrimitiveTypedTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${packageName}${returnType} {
	return ${packageName}${returnType}(ms.getOrig().${originFieldName})
}

// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${packageName}${returnType}) {
	ms.getOrig().${originFieldName} = ${rawType}(v)
}`

const accessorsPrimitiveTypedTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	assert.Equal(t, ${packageName}${returnType}(${defaultVal}), ms.${fieldName}())
	testVal${fieldName} := ${packageName}${returnType}(${testValue})
	ms.Set${fieldName}(testVal${fieldName})
	assert.Equal(t, testVal${fieldName}, ms.${fieldName}())
}`

const accessorsPrimitiveSliceTestTemplate = `func Test${structName}_${fieldName}(t *testing.T) {
	ms := New${structName}()
	assert.Equal(t, ${defaultVal}, ms.${fieldName}().AsRaw())
	ms.${fieldName}().FromRaw(${testValue})
	assert.Equal(t, ${testValue}, ms.${fieldName}().AsRaw())
}`

const accessorsOptionalPrimitiveValueTemplate = `// ${fieldName} returns the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) ${fieldName}() ${returnType} {
	return ms.getOrig().Get${fieldName}()
}
// Has${fieldName} returns true if the ${structName} contains a
// ${fieldName} value, false otherwise.
func (ms ${structName}) Has${fieldName}() bool {
	return ms.getOrig().${fieldName}_ != nil
}
// Set${fieldName} replaces the ${lowerFieldName} associated with this ${structName}.
func (ms ${structName}) Set${fieldName}(v ${returnType}) {
	ms.getOrig().${fieldName}_ = &${originStructType}{${fieldName}: v}
}`

type baseField interface {
	generateAccessors(ms baseStruct, sb *bytes.Buffer)

	generateAccessorsTest(ms baseStruct, sb *bytes.Buffer)

	generateSetWithTestValue(sb *bytes.Buffer)

	generateCopyToValue(ms baseStruct, sb *bytes.Buffer)
}

type sliceField struct {
	fieldName   string
	returnSlice baseSlice
}

func (sf *sliceField) generateAccessors(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return sf.fieldName
		case "packageName":
			if sf.returnSlice.getPackageName() != ms.getPackageName() {
				return sf.returnSlice.getPackageName() + "."
			}
			return ""
		case "returnType":
			return sf.returnSlice.getName()
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateAccessorsTest(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsSliceTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return sf.fieldName
		case "packageName":
			if sf.returnSlice.getPackageName() != ms.getPackageName() {
				return sf.returnSlice.getPackageName() + "."
			}
			return ""
		case "returnType":
			return sf.returnSlice.getName()
		default:
			panic(name)
		}
	}))
}

func (sf *sliceField) generateSetWithTestValue(sb *bytes.Buffer) {
	sb.WriteString("\tFillTest" + sf.returnSlice.getName() + "(New" + sf.returnSlice.getName() + "(&tv.orig." + sf.fieldName + "))")
}

func (sf *sliceField) generateCopyToValue(_ baseStruct, sb *bytes.Buffer) {
	sb.WriteString("\tms." + sf.fieldName + "().CopyTo(dest." + sf.fieldName + "())")
}

var _ baseField = (*sliceField)(nil)

type messageValueField struct {
	fieldName     string
	returnMessage baseStruct
}

func (mf *messageValueField) generateAccessors(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsMessageValueTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return mf.fieldName
		case "lowerFieldName":
			return strings.ToLower(mf.fieldName)
		case "packageName":
			if mf.returnMessage.getPackageName() != ms.getPackageName() {
				return mf.returnMessage.getPackageName() + "."
			}
			return ""
		case "returnType":
			return mf.returnMessage.getName()
		default:
			panic(name)
		}
	}))
}

func (mf *messageValueField) generateAccessorsTest(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsMessageValueTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return mf.fieldName
		case "returnType":
			return mf.returnMessage.getName()
		case "packageName":
			if mf.returnMessage.getPackageName() != ms.getPackageName() {
				return mf.returnMessage.getPackageName() + "."
			}
			return ""
		default:
			panic(name)
		}
	}))
}

func (mf *messageValueField) generateSetWithTestValue(sb *bytes.Buffer) {
	sb.WriteString("\tFillTest" + mf.returnMessage.getName() + "(New" + mf.returnMessage.getName() + "(&tv.orig." + mf.fieldName + "))")
}

func (mf *messageValueField) generateCopyToValue(_ baseStruct, sb *bytes.Buffer) {
	sb.WriteString("\tms." + mf.fieldName + "().CopyTo(dest." + mf.fieldName + "())")
}

var _ baseField = (*messageValueField)(nil)

type primitiveField struct {
	fieldName  string
	returnType string
	defaultVal string
	testVal    string
}

func (pf *primitiveField) generateAccessors(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsPrimitiveTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "packageName":
			return ""
		case "fieldName":
			return pf.fieldName
		case "lowerFieldName":
			return strings.ToLower(pf.fieldName)
		case "returnType":
			return pf.returnType
		default:
			panic(name)
		}
	}))
}

func (pf *primitiveField) generateAccessorsTest(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "packageName":
			return ""
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

func (pf *primitiveField) generateSetWithTestValue(sb *bytes.Buffer) {
	sb.WriteString("\ttv.orig." + pf.fieldName + " = " + pf.testVal)
}

func (pf *primitiveField) generateCopyToValue(_ baseStruct, sb *bytes.Buffer) {
	sb.WriteString("\tdest.Set" + pf.fieldName + "(ms." + pf.fieldName + "())")
}

var _ baseField = (*primitiveField)(nil)

type primitiveType struct {
	structName  string
	packageName string
	rawType     string
	defaultVal  string
	testVal     string
}

// Types that has defined a custom type (e.g. "type Timestamp uint64")
type primitiveTypedField struct {
	fieldName       string
	originFieldName string
	returnType      *primitiveType
}

func (ptf *primitiveTypedField) generateAccessors(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsPrimitiveTypedTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return ptf.fieldName
		case "lowerFieldName":
			return strings.ToLower(ptf.fieldName)
		case "returnType":
			return ptf.returnType.structName
		case "packageName":
			if ptf.returnType.packageName != ms.getPackageName() {
				return ptf.returnType.packageName + "."
			}
			return ""
		case "rawType":
			return ptf.returnType.rawType
		case "originFieldName":
			if ptf.originFieldName == "" {
				return ptf.fieldName
			}
			return ptf.originFieldName
		default:
			panic(name)
		}
	}))
}

func (ptf *primitiveTypedField) generateAccessorsTest(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsPrimitiveTypedTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "defaultVal":
			return ptf.returnType.defaultVal
		case "packageName":
			if ptf.returnType.packageName != ms.getPackageName() {
				return ptf.returnType.packageName + "."
			}
			return ""
		case "returnType":
			return ptf.returnType.structName
		case "fieldName":
			return ptf.fieldName
		case "testValue":
			return ptf.returnType.testVal
		default:
			panic(name)
		}
	}))
}

func (ptf *primitiveTypedField) generateSetWithTestValue(sb *bytes.Buffer) {
	originFieldName := ptf.fieldName
	if ptf.originFieldName != "" {
		originFieldName = ptf.originFieldName
	}
	sb.WriteString("\ttv.orig." + originFieldName + " = " + ptf.returnType.testVal)
}

func (ptf *primitiveTypedField) generateCopyToValue(_ baseStruct, sb *bytes.Buffer) {
	sb.WriteString("\tdest.Set" + ptf.fieldName + "(ms." + ptf.fieldName + "())")
}

var _ baseField = (*primitiveTypedField)(nil)

// primitiveSliceField is used to generate fields for slice of primitive types
type primitiveSliceField struct {
	fieldName         string
	returnPackageName string
	returnType        string
	defaultVal        string
	rawType           string
	testVal           string
}

func (psf *primitiveSliceField) generateAccessors(ms baseStruct, sb *bytes.Buffer) {
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
		case "packageName":
			if psf.returnPackageName != ms.getPackageName() {
				return psf.returnPackageName + "."
			}
			return ""
		default:
			panic(name)
		}
	}))
}

func (psf *primitiveSliceField) generateAccessorsTest(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsPrimitiveSliceTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "packageName":
			if psf.returnPackageName != ms.getPackageName() {
				return psf.returnPackageName + "."
			}
			return ""
		case "returnType":
			return psf.returnType
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

func (psf *primitiveSliceField) generateSetWithTestValue(sb *bytes.Buffer) {
	sb.WriteString("\ttv.orig." + psf.fieldName + " = " + psf.testVal)
}

func (psf *primitiveSliceField) generateCopyToValue(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString("\tms." + psf.fieldName + "().CopyTo(dest." + psf.fieldName + "())")
}

var _ baseField = (*primitiveSliceField)(nil)

type oneOfField struct {
	originTypePrefix           string
	originFieldName            string
	typeName                   string
	testValueIdx               int
	values                     []oneOfValue
	omitOriginFieldNameInNames bool
}

func (of *oneOfField) generateAccessors(ms baseStruct, sb *bytes.Buffer) {
	of.generateTypeAccessors(ms, sb)
	sb.WriteString("\n")
	for _, v := range of.values {
		v.generateAccessors(ms, of, sb)
		sb.WriteString("\n")
	}
}

func (of *oneOfField) generateTypeAccessors(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(oneOfTypeAccessorHeaderTemplate, func(name string) string {
		switch name {
		case "lowerOriginFieldName":
			return strings.ToLower(of.originFieldName)
		case "typeFuncName":
			return of.typeFuncName()
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
	sb.WriteString("\treturn " + of.typeName + "Empty\n")
	sb.WriteString("}\n")
}

func (of *oneOfField) typeFuncName() string {
	const typeSuffix = "Type"
	if of.omitOriginFieldNameInNames {
		return typeSuffix
	}
	return of.originFieldName + typeSuffix
}

func (of *oneOfField) generateAccessorsTest(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(oneOfTypeAccessorHeaderTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "typeFuncName":
			return of.typeFuncName()
		case "typeName":
			return of.typeName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
	for _, v := range of.values {
		v.generateTests(ms, of, sb)
		sb.WriteString("\n")
	}
}

func (of *oneOfField) generateSetWithTestValue(sb *bytes.Buffer) {
	of.values[of.testValueIdx].generateSetWithTestValue(of, sb)
}

func (of *oneOfField) generateCopyToValue(_ baseStruct, sb *bytes.Buffer) {
	sb.WriteString("\tswitch ms." + of.typeFuncName() + "() {\n")
	for _, v := range of.values {
		v.generateCopyToValue(of, sb)
	}
	sb.WriteString("\t}\n")
}

var _ baseField = (*oneOfField)(nil)

type oneOfValue interface {
	generateAccessors(ms baseStruct, of *oneOfField, sb *bytes.Buffer)
	generateTests(ms baseStruct, of *oneOfField, sb *bytes.Buffer)
	generateSetWithTestValue(of *oneOfField, sb *bytes.Buffer)
	generateCopyToValue(of *oneOfField, sb *bytes.Buffer)
	generateTypeSwitchCase(of *oneOfField, sb *bytes.Buffer)
}

type oneOfPrimitiveValue struct {
	fieldName       string
	defaultVal      string
	testVal         string
	returnType      string
	originFieldName string
}

func (opv *oneOfPrimitiveValue) generateAccessors(ms baseStruct, of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsOneOfPrimitiveTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "accessorFieldName":
			return opv.accessorFieldName(of)
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

func (opv *oneOfPrimitiveValue) generateTests(ms baseStruct, of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsOneOfPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "defaultVal":
			return opv.defaultVal
		case "packageName":
			return ""
		case "accessorFieldName":
			return opv.accessorFieldName(of)
		case "testValue":
			return opv.testVal
		case "originOneOfTypeFuncName":
			return of.typeFuncName()
		case "typeName":
			return of.typeName + opv.fieldName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (opv *oneOfPrimitiveValue) accessorFieldName(of *oneOfField) string {
	if of.omitOriginFieldNameInNames {
		return opv.fieldName
	}
	return opv.fieldName + of.originFieldName
}

func (opv *oneOfPrimitiveValue) generateSetWithTestValue(of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString("\ttv.orig." + of.originFieldName + " = &" + of.originTypePrefix + opv.originFieldName + "{" + opv.originFieldName + ":" + opv.testVal + "}")
}

func (opv *oneOfPrimitiveValue) generateCopyToValue(of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString("\tcase " + of.typeName + opv.fieldName + ":\n")
	sb.WriteString("\tdest.Set" + opv.accessorFieldName(of) + "(ms." + opv.accessorFieldName(of) + "())\n")
}

func (opv *oneOfPrimitiveValue) generateTypeSwitchCase(of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString("\tcase *" + of.originTypePrefix + opv.originFieldName + ":\n")
	sb.WriteString("\t\treturn " + of.typeName + opv.fieldName + "\n")
}

var _ oneOfValue = (*oneOfPrimitiveValue)(nil)

type oneOfMessageValue struct {
	fieldName              string
	originFieldPackageName string
	returnMessage          *messageValueStruct
}

func (omv *oneOfMessageValue) generateAccessors(ms baseStruct, of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsOneOfMessageTemplate, func(name string) string {
		switch name {
		case "fieldName":
			return omv.fieldName
		case "lowerFieldName":
			return strings.ToLower(omv.fieldName)
		case "originOneOfTypeFuncName":
			return of.typeFuncName()
		case "originOneOfFieldName":
			return of.originFieldName
		case "originFieldPackageName":
			return omv.originFieldPackageName
		case "originStructType":
			return of.originTypePrefix + omv.fieldName
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

func (omv *oneOfMessageValue) generateTests(ms baseStruct, of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsOneOfMessageTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "fieldName":
			return omv.fieldName
		case "returnType":
			return omv.returnMessage.structName
		case "originOneOfTypeFuncName":
			return of.typeFuncName()
		case "typeName":
			return of.typeName + omv.returnMessage.structName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (omv *oneOfMessageValue) generateSetWithTestValue(of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString("\ttv.orig." + of.originFieldName + " = &" + of.originTypePrefix + omv.fieldName + "{" + omv.fieldName + ": &" + omv.originFieldPackageName + "." + omv.fieldName + "{}}\n")
	sb.WriteString("\tFillTest" + omv.returnMessage.structName + "(New" + omv.fieldName + "(tv.orig.Get" + omv.fieldName + "()))")
}

func (omv *oneOfMessageValue) generateCopyToValue(of *oneOfField, sb *bytes.Buffer) {
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

func (omv *oneOfMessageValue) generateTypeSwitchCase(of *oneOfField, sb *bytes.Buffer) {
	sb.WriteString("\tcase *" + of.originTypePrefix + omv.fieldName + ":\n")
	sb.WriteString("\t\treturn " + of.typeName + omv.fieldName + "\n")
}

var _ oneOfValue = (*oneOfMessageValue)(nil)

type optionalPrimitiveValue struct {
	fieldName        string
	defaultVal       string
	testVal          string
	returnType       string
	originTypePrefix string
}

func (opv *optionalPrimitiveValue) generateAccessors(ms baseStruct, sb *bytes.Buffer) {
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
		case "originStructType":
			return opv.originTypePrefix + opv.fieldName
		default:
			panic(name)
		}
	}))
	sb.WriteString("\n")
}

func (opv *optionalPrimitiveValue) generateAccessorsTest(ms baseStruct, sb *bytes.Buffer) {
	sb.WriteString(os.Expand(accessorsPrimitiveTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.getName()
		case "packageName":
			return ""
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

func (opv *optionalPrimitiveValue) generateSetWithTestValue(sb *bytes.Buffer) {
	sb.WriteString("\ttv.orig." + opv.fieldName + "_ = &" + opv.originTypePrefix + opv.fieldName + "{" + opv.fieldName + ":" + opv.testVal + "}")
}

func (opv *optionalPrimitiveValue) generateCopyToValue(_ baseStruct, sb *bytes.Buffer) {
	sb.WriteString("if ms.Has" + opv.fieldName + "(){\n")
	sb.WriteString("\tdest.Set" + opv.fieldName + "(ms." + opv.fieldName + "())\n")
	sb.WriteString("}\n")
}

var _ baseField = (*optionalPrimitiveValue)(nil)
