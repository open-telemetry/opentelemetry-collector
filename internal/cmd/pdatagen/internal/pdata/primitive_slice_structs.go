// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"strings"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

// primitiveSliceStruct generates a struct for a slice of primitive value elements. The structs are always generated
// in a way that they can be used as fields in structs from other packages (using the internal package).
type primitiveSliceStruct struct {
	structName  string
	packageName string
	itemType    string

	testOrigVal          string
	testInterfaceOrigVal []any
	testSetVal           string
	testNewVal           string
}

func (iss *primitiveSliceStruct) getName() string {
	return iss.structName
}

func (iss *primitiveSliceStruct) getPackageName() string {
	return iss.packageName
}

func (iss *primitiveSliceStruct) generate(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(primitiveSliceTemplate, iss.templateFields(packageInfo)))
}

func (iss *primitiveSliceStruct) generateTests(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(primitiveSliceTestTemplate, iss.templateFields(packageInfo)))
}

func (iss *primitiveSliceStruct) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(primitiveSliceInternalTemplate, iss.templateFields(packageInfo)))
}

func (iss *primitiveSliceStruct) getOriginName() string {
	return iss.getName()
}

func (iss *primitiveSliceStruct) getOriginFullName() string {
	return iss.getName()
}

func (iss *primitiveSliceStruct) getHasWrapper() bool {
	return usedByOtherDataTypes(iss.packageName)
}

func (iss *primitiveSliceStruct) getHasOnlyInternal() bool {
	return false
}

func (iss *primitiveSliceStruct) getElementOriginName() string {
	return upperFirst(iss.itemType)
}

func (iss *primitiveSliceStruct) getElementNullable() bool {
	return false
}

func (iss *primitiveSliceStruct) getProtoMessage() *proto.Message {
	return nil
}

func (iss *primitiveSliceStruct) templateFields(packageInfo *PackageInfo) map[string]any {
	return map[string]any{
		"structName":           iss.getName(),
		"itemType":             iss.itemType,
		"elementOriginName":    iss.getElementOriginName(),
		"lowerStructName":      strings.ToLower(iss.structName[:1]) + iss.structName[1:],
		"testOrigVal":          iss.testOrigVal,
		"testInterfaceOrigVal": iss.testInterfaceOrigVal,
		"testSetVal":           iss.testSetVal,
		"testNewVal":           iss.testNewVal,
		"packageName":          packageInfo.name,
		"imports":              packageInfo.imports,
		"testImports":          packageInfo.testImports,
	}
}

func upperFirst(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}
