// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/template"
)

type baseStruct interface {
	getName() string
	getOriginName() string
	getOriginFullName() string
	getHasWrapper() bool
	getHasOnlyOrig() bool
	generate(packageInfo *PackageInfo) []byte
	generateTests(packageInfo *PackageInfo) []byte
	generateInternal(packageInfo *PackageInfo) []byte
	generateInternalTests(packageInfo *PackageInfo) []byte
}

// messageStruct generates a struct for a proto message. The struct can be generated both as a common struct
// that can be used as a field in struct from other packages and as an isolated struct with depending on a package name.
type messageStruct struct {
	structName     string
	packageName    string
	description    string
	originFullName string
	fields         []Field
	hasWrapper     bool
	hasOnlyOrig    bool
}

func (ms *messageStruct) getName() string {
	return ms.structName
}

func (ms *messageStruct) generate(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(messageTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageStruct) generateTests(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(messageTestTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageStruct) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(messageInternalTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageStruct) generateInternalTests(packageInfo *PackageInfo) []byte {
	return []byte(template.Execute(messageInternalTestTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageStruct) templateFields(packageInfo *PackageInfo) map[string]any {
	hasWrapper := ms.hasWrapper
	if !hasWrapper {
		hasWrapper = usedByOtherDataTypes(ms.packageName)
	}
	return map[string]any{
		"messageStruct":  ms,
		"fields":         ms.fields,
		"structName":     ms.getName(),
		"originFullName": ms.getOriginFullName(),
		"originName":     ms.getOriginName(),
		"description":    ms.description,
		"hasWrapper":     hasWrapper,
		"origAccessor":   origAccessor(hasWrapper),
		"stateAccessor":  stateAccessor(hasWrapper),
		"packageName":    packageInfo.name,
		"imports":        packageInfo.imports,
		"testImports":    packageInfo.testImports,
	}
}

func (ms *messageStruct) getHasWrapper() bool {
	if ms.hasWrapper {
		return true
	}
	if ms.hasOnlyOrig {
		return false
	}
	return usedByOtherDataTypes(ms.packageName)
}

func (ms *messageStruct) getHasOnlyOrig() bool {
	return ms.hasOnlyOrig
}

func (ms *messageStruct) getOriginName() string {
	return proto.ExtractNameFromFull(ms.originFullName)
}

func (ms *messageStruct) getOriginFullName() string {
	return ms.originFullName
}

var _ baseStruct = (*messageStruct)(nil)
