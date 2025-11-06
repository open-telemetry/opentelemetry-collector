// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"path/filepath"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

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
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		entityRefSlice,
		entityRef,
	},
}

var entityRefSlice = &messageSlice{
	structName:      "EntityRefSlice",
	packageName:     "entity",
	elementNullable: true,
	element:         entityRef,
}

var entityRef = &messageStruct{
	structName:    "EntityRef",
	packageName:   "entity",
	protoName:     "EntityRef",
	upstreamProto: "gootlpcommon.EntityRef",
	fields: []Field{
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   1,
			protoType: proto.TypeString,
		},
		&PrimitiveField{
			fieldName: "Type",
			protoID:   2,
			protoType: proto.TypeString,
		},
		&SliceField{
			fieldName:   "IdKeys",
			protoID:     3,
			protoType:   proto.TypeString,
			returnSlice: stringSlice,
		},
		&SliceField{
			fieldName:   "DescriptionKeys",
			protoID:     4,
			protoType:   proto.TypeString,
			returnSlice: stringSlice,
		},
	},
}
