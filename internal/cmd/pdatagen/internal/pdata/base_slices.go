// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

type baseSlice interface {
	getName() string
	getHasWrapper() bool
	getOriginFullName() string
	getElementOriginName() string
	getElementNullable() bool
	getPackageName() string
}

// messageSlice generates code for a slice of pointer fields. The generated structs cannot be used from other packages.
type messageSlice struct {
	structName      string
	packageName     string
	elementNullable bool
	element         *messageStruct
}

func (ss *messageSlice) getProtoMessage() *proto.Message {
	return nil
}

func (ss *messageSlice) getName() string {
	return ss.structName
}

func (ss *messageSlice) getPackageName() string {
	return ss.packageName
}

func (ss *messageSlice) generate(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(sliceTemplate, ss.templateFields(packageInfo)))
}

func (ss *messageSlice) generateTests(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(sliceTestTemplate, ss.templateFields(packageInfo)))
}

func (ss *messageSlice) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(sliceInternalTemplate, ss.templateFields(packageInfo)))
}

func (ss *messageSlice) templateFields(packageInfo *PackageInfo) map[string]any {
	hasWrapper := usedByOtherDataTypes(ss.packageName)
	return map[string]any{
		"hasWrapper":        usedByOtherDataTypes(ss.packageName),
		"structName":        ss.structName,
		"elementName":       ss.element.getName(),
		"elementOriginName": ss.getElementOriginName(),
		"elementNullable":   ss.elementNullable,
		"origAccessor":      origAccessor(hasWrapper),
		"stateAccessor":     stateAccessor(hasWrapper),
		"packageName":       packageInfo.name,
		"imports":           packageInfo.imports,
		"testImports":       packageInfo.testImports,
	}
}

func (ss *messageSlice) getOriginName() string {
	return ss.element.getOriginName() + "Slice"
}

func (ss *messageSlice) getOriginFullName() string {
	return ss.element.getOriginFullName()
}

func (ss *messageSlice) getHasWrapper() bool {
	return usedByOtherDataTypes(ss.packageName)
}

func (ss *messageSlice) getHasOnlyInternal() bool {
	return false
}

func (ss *messageSlice) getElementOriginName() string {
	return ss.element.getOriginName()
}

func (ss *messageSlice) getElementNullable() bool {
	return ss.elementNullable
}

var _ baseStruct = (*messageSlice)(nil)
