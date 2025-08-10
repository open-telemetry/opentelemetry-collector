// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"
import (
	"strings"
)

type baseStruct interface {
	getName() string
	getOriginName() string
	getHasWrapper() bool
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
}

func (ms *messageStruct) getName() string {
	return ms.structName
}

func (ms *messageStruct) generate(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(messageTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageStruct) generateTests(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(messageTestTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageStruct) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(messageInternalTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageStruct) generateInternalTests(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(messageInternalTestTemplate, ms.templateFields(packageInfo)))
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
		"originFullName": ms.originFullName,
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
	return usedByOtherDataTypes(ms.packageName)
}

func (ms *messageStruct) getOriginName() string {
	_, after, _ := strings.Cut(ms.originFullName, ".")
	return after
}

var _ baseStruct = (*messageStruct)(nil)
