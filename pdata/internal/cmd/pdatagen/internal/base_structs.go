// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"strings"
)

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
	// Create filtered package info for internal files - add necessary imports based on origin type
	var imports []string

	// Add necessary imports based on the origin type
	if strings.Contains(ms.originFullName, "otlpcommon.") {
		imports = append(imports, `otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"`)
	}
	if strings.Contains(ms.originFullName, "otlpresource.") {
		imports = append(imports, `otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"`)
	}

	internalPackageInfo := &PackageInfo{
		name:        "internal",
		path:        "internal",
		imports:     imports,
		testImports: packageInfo.testImports,
	}
	return []byte(executeTemplate(messageInternalTemplate, ms.templateFields(internalPackageInfo)))
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
