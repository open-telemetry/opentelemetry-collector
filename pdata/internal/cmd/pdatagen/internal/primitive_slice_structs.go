// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"bytes"
	"strings"
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
	var sb bytes.Buffer
	if err := primitiveSliceTemplate.Execute(&sb, iss.templateFields(packageInfo)); err != nil {
		panic(err)
	}
	return sb.Bytes()
}

func (iss *primitiveSliceStruct) generateTests(packageInfo *PackageInfo) []byte {
	var sb bytes.Buffer
	if err := primitiveSliceTestTemplate.Execute(&sb, iss.templateFields(packageInfo)); err != nil {
		panic(err)
	}
	return sb.Bytes()
}

func (iss *primitiveSliceStruct) generateInternal(packageInfo *PackageInfo) []byte {
	var sb bytes.Buffer
	if err := primitiveSliceInternalTemplate.Execute(&sb, iss.templateFields(packageInfo)); err != nil {
		panic(err)
	}
	return sb.Bytes()
}

func (iss *primitiveSliceStruct) templateFields(packageInfo *PackageInfo) map[string]any {
	return map[string]any{
		"structName":           iss.structName,
		"itemType":             iss.itemType,
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
