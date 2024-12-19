// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"
import (
	"path/filepath"
)

var pentityotlp = &Package{
	info: &PackageInfo{
		name: "pentityotlp",
		path: filepath.Join("pentity", "pentityotlp"),
		imports: []string{
			`otlpcollectorentity "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/entities/v1"`,
		},
		testImports: []string{
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
		},
	},
	structs: []baseStruct{
		exportEntitiesPartialSuccess,
	},
}

var exportEntitiesPartialSuccess = &messageValueStruct{
	structName:     "ExportPartialSuccess",
	description:    "// ExportPartialSuccess represents the details of a partially successful export request.",
	originFullName: "otlpcollectorentity.ExportEntitiesPartialSuccess",
	fields: []baseField{
		&primitiveField{
			fieldName:  "RejectedEntities",
			returnType: "int64",
			defaultVal: `int64(0)`,
			testVal:    `int64(13)`,
		},
		&primitiveField{
			fieldName:  "ErrorMessage",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"error message"`,
		},
	},
}
