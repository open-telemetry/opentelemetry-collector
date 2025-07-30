// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"bytes"
)

type baseSlice interface {
	getName() string
	getPackageName() string
}

// sliceOfPtrs generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfPtrs struct {
	structName  string
	packageName string
	element     *messageStruct
}

func (ss *sliceOfPtrs) getName() string {
	return ss.structName
}

func (ss *sliceOfPtrs) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfPtrs) generate(packageInfo *PackageInfo) []byte {
	var sb bytes.Buffer
	if err := sliceTemplate.Execute(&sb, ss.templateFields(packageInfo)); err != nil {
		panic(err)
	}
	return sb.Bytes()
}

func (ss *sliceOfPtrs) generateTests(packageInfo *PackageInfo) []byte {
	var sb bytes.Buffer
	if err := sliceTestTemplate.Execute(&sb, ss.templateFields(packageInfo)); err != nil {
		panic(err)
	}
	return sb.Bytes()
}

func (ss *sliceOfPtrs) templateFields(packageInfo *PackageInfo) map[string]any {
	orig := origAccessor(ss.packageName)
	state := stateAccessor(ss.packageName)
	return map[string]any{
		"type":               "sliceOfPtrs",
		"isCommon":           usedByOtherDataTypes(ss.packageName),
		"structName":         ss.structName,
		"elementName":        ss.element.getName(),
		"originName":         ss.element.originFullName,
		"originElementType":  "*" + ss.element.originFullName,
		"originElementPtr":   "",
		"emptyOriginElement": "&" + ss.element.originFullName + "{}",
		"newElement":         "new" + ss.element.getName() + "((*es." + orig + ")[i], es." + state + ")",
		"origAccessor":       orig,
		"stateAccessor":      state,
		"packageName":        packageInfo.name,
		"imports":            packageInfo.imports,
		"testImports":        packageInfo.testImports,
	}
}

func (ss *sliceOfPtrs) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceInternalTemplate, ss.templateFields(packageInfo)))
}

var _ baseStruct = (*sliceOfPtrs)(nil)

// sliceOfValues generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type sliceOfValues struct {
	structName  string
	packageName string
	element     *messageStruct
}

func (ss *sliceOfValues) getName() string {
	return ss.structName
}

func (ss *sliceOfValues) getPackageName() string {
	return ss.packageName
}

func (ss *sliceOfValues) generate(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfValues) generateTests(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceTestTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfValues) templateFields(packageInfo *PackageInfo) map[string]any {
	orig := origAccessor(ss.packageName)
	state := stateAccessor(ss.packageName)
	return map[string]any{
		"type":               "sliceOfValues",
		"isCommon":           usedByOtherDataTypes(ss.packageName),
		"structName":         ss.getName(),
		"elementName":        ss.element.getName(),
		"originName":         ss.element.originFullName,
		"originElementType":  ss.element.originFullName,
		"originElementPtr":   "&",
		"emptyOriginElement": ss.element.originFullName + "{}",
		"newElement":         "new" + ss.element.getName() + "(&(*es." + orig + ")[i], es." + state + ")",
		"origAccessor":       orig,
		"stateAccessor":      state,
		"packageName":        packageInfo.name,
		"imports":            packageInfo.imports,
		"testImports":        packageInfo.testImports,
	}
}

func (ss *sliceOfValues) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceInternalTemplate, ss.templateFields(packageInfo)))
}

var _ baseStruct = (*sliceOfValues)(nil)
