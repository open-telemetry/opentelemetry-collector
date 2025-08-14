// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import "path/filepath"

var xpdataEntity = &Package{
	info: &PackageInfo{
		name: "entity",
		path: filepath.Join("xpdata", "entity"),
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			``,
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
			`gootlpcommon "go.opentelemetry.io/proto/slim/otlp/common/v1"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"`,
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
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   1,
			protoType: ProtoTypeString,
		},
		&PrimitiveField{
			fieldName: "Type",
			protoID:   2,
			protoType: ProtoTypeString,
		},
		&SliceField{
			fieldName:   "IdKeys",
			protoID:     3,
			protoType:   ProtoTypeString,
			returnSlice: stringSlice,
		},
		&SliceField{
			fieldName:   "DescriptionKeys",
			protoID:     4,
			protoType:   ProtoTypeString,
			returnSlice: stringSlice,
		},
	},
}
