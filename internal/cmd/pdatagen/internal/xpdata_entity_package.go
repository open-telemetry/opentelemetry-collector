// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import "path/filepath"

var xpdataEntity = &Package{
	info: &PackageInfo{
		name: "entity",
		path: filepath.Join("xpdata", "entity"),
		imports: []string{
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
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

var entityRef = &messageStruct{
	structName:     "EntityRef",
	packageName:    "entity",
	originFullName: "otlpcommon.EntityRef",
	fields: []Field{
		schemaURLField,
		&PrimitiveField{
			fieldName: "Type",
			protoType: ProtoTypeString,
		},
		&SliceField{
			fieldName:   "IdKeys",
			returnSlice: stringSlice,
		},
		&SliceField{
			fieldName:   "DescriptionKeys",
			returnSlice: stringSlice,
		},
	},
}
