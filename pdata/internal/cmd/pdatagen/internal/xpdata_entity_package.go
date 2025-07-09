// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import "path/filepath"

var xpdataEntity = &Package{
	info: &PackageInfo{
		name: "entity",
		path: filepath.Join("xpdata", "entity"),
		imports: []string{
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
			`otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"`,
			`otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"`,
		},
		testImports: []string{
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		entityRefSlice,
		entityRef,
	},
}

var entityRefSlice = &sliceOfPtrs{
	structName:  "EntityRefSlice",
	packageName: "entity",
	element:     entityRef,
}

var entityRef = &messageValueStruct{
	structName:     "EntityRef",
	packageName:    "entity",
	originFullName: "otlpcommon.EntityRef",
	fields: []baseField{
		schemaURLField,
		&primitiveField{
			fieldName:  "Type",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"host"`,
		},
		&sliceField{
			fieldName:   "IdKeys",
			returnSlice: stringSlice,
		},
		&sliceField{
			fieldName:   "DescriptionKeys",
			returnSlice: stringSlice,
		},
	},
}
