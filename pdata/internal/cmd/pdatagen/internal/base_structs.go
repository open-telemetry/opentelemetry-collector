// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

type baseStruct interface {
	getName() string
	generate(packageInfo *PackageInfo) []byte
	generateTests(packageInfo *PackageInfo) []byte
	generateInternal(packageInfo *PackageInfo) []byte
}

// messageValueStruct generates a struct for a proto message. The struct can be generated both as a common struct
// that can be used as a field in struct from other packages and as an isolated struct with depending on a package name.
type messageValueStruct struct {
	structName     string
	packageName    string
	description    string
	originFullName string
	fields         []baseField
}

func (ms *messageValueStruct) getName() string {
	return ms.structName
}

func (ms *messageValueStruct) generate(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(messageTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageValueStruct) generateTests(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(messageTestTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageValueStruct) generateInternal(packageInfo *PackageInfo) []byte {
	return []byte(executeTemplate(messageInternalTemplate, ms.templateFields(packageInfo)))
}

func (ms *messageValueStruct) templateFields(packageInfo *PackageInfo) map[string]any {
	return map[string]any{
		"messageStruct": ms,
		"fields":        ms.fields,
		"structName":    ms.structName,
		"originName":    ms.originFullName,
		"description":   ms.description,
		"isCommon":      usedByOtherDataTypes(ms.packageName),
		"origAccessor":  origAccessor(ms.packageName),
		"stateAccessor": stateAccessor(ms.packageName),
		"packageName":   packageInfo.name,
		"imports":       packageInfo.imports,
		"testImports":   packageInfo.testImports,
	}
}

var _ baseStruct = (*messageValueStruct)(nil)
