// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

type baseSlice interface {
	getName() string
	getOriginName() string
	getHasWrapper() bool
	getElementProtoType() ProtoType
	getElementOriginName() string
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
	return []byte(executeTemplate(sliceTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfPtrs) generateTests(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceTestTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfPtrs) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceInternalTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfPtrs) generateInternalTests(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceInternalTestTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfPtrs) templateFields(packageInfo *PackageInfo) map[string]any {
	hasWrapper := usedByOtherDataTypes(ss.packageName)
	orig := origAccessor(hasWrapper)
	state := stateAccessor(hasWrapper)
	return map[string]any{
		"type":                  "sliceOfPtrs",
		"hasWrapper":            usedByOtherDataTypes(ss.packageName),
		"structName":            ss.structName,
		"elementName":           ss.element.getName(),
		"elementOriginFullName": ss.element.originFullName,
		"elementOriginName":     ss.getElementOriginName(),
		"originElementType":     "*" + ss.element.originFullName,
		"originElementPtr":      "",
		"emptyOriginElement":    "&" + ss.element.originFullName + "{}",
		"newElement":            "new" + ss.element.getName() + "((*es." + orig + ")[i], es." + state + ")",
		"origAccessor":          origAccessor(hasWrapper),
		"stateAccessor":         stateAccessor(hasWrapper),
		"packageName":           packageInfo.name,
		"imports":               packageInfo.imports,
		"testImports":           packageInfo.testImports,
	}
}

func (ss *sliceOfPtrs) getOriginName() string {
	return ss.element.getOriginName() + "Slice"
}

func (ss *sliceOfPtrs) getElementProtoType() ProtoType {
	return ProtoTypeMessage
}

func (ss *sliceOfPtrs) getHasWrapper() bool {
	return usedByOtherDataTypes(ss.packageName)
}

func (ss *sliceOfPtrs) getElementOriginName() string {
	return ss.element.getOriginName()
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

func (ss *sliceOfValues) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceInternalTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfValues) generateInternalTests(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(sliceInternalTestTemplate, ss.templateFields(packageInfo)))
}

func (ss *sliceOfValues) templateFields(packageInfo *PackageInfo) map[string]any {
	hasWrapper := usedByOtherDataTypes(ss.packageName)
	orig := origAccessor(hasWrapper)
	state := stateAccessor(hasWrapper)
	return map[string]any{
		"type":                  "sliceOfValues",
		"hasWrapper":            usedByOtherDataTypes(ss.packageName),
		"structName":            ss.getName(),
		"elementName":           ss.element.getName(),
		"elementOriginFullName": ss.element.originFullName,
		"elementOriginName":     ss.getElementOriginName(),
		"originElementType":     ss.element.originFullName,
		"originElementPtr":      "&",
		"emptyOriginElement":    ss.element.originFullName + "{}",
		"newElement":            "new" + ss.element.getName() + "(&(*es." + orig + ")[i], es." + state + ")",
		"origAccessor":          orig,
		"stateAccessor":         state,
		"packageName":           packageInfo.name,
		"imports":               packageInfo.imports,
		"testImports":           packageInfo.testImports,
	}
}

func (ss *sliceOfValues) getOriginName() string {
	return ss.element.getOriginName() + "Slice"
}

func (ss *sliceOfValues) getElementProtoType() ProtoType {
	return ProtoTypeMessage
}

func (ss *sliceOfValues) getHasWrapper() bool {
	return usedByOtherDataTypes(ss.packageName)
}

func (ss *sliceOfValues) getElementOriginName() string {
	return ss.element.getOriginName()
}

var _ baseSlice = (*sliceOfValues)(nil)
