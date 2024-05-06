// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"
import (
	"path/filepath"
)

var pprofileotlp = &Package{
	name: "pprofileotlp",
	path: filepath.Join("pprofile", "pprofileotlp"),
	imports: []string{
		`otlpcollectorprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1experimental"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
	},
	structs: []baseStruct{
		exportProfilesPartialSuccess,
	},
}

var exportProfilesPartialSuccess = &messageValueStruct{
	structName:     "ExportPartialSuccess",
	description:    "// ExportPartialSuccess represents the details of a partially successful export request.",
	originFullName: "otlpcollectorprofile.ExportProfilesPartialSuccess",
	fields: []baseField{
		&primitiveField{
			fieldName:  "RejectedProfiles",
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
