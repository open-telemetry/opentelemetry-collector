// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

type baseStruct interface {
	getName() string
	generate(packageInfo *PackageInfo) []byte
	generateTests(packageInfo *PackageInfo) []byte
	generateInternal(packageInfo *PackageInfo) []byte
}

// messageStruct generates a struct for a proto message. The struct can be generated both as a common struct
// that can be used as a field in struct from other packages and as an isolated struct with depending on a package name.
type messageStruct struct {
	structName     string
	packageName    string
	description    string
	originFullName string
	fields         []Field
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

func (ms *messageStruct) templateFields(packageInfo *PackageInfo) map[string]any {
	return map[string]any{
		"messageStruct": ms,
		"fields":        ms.fields,
		"structName":    ms.getName(),
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

var _ baseStruct = (*messageStruct)(nil)
