// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"
import (
	"path/filepath"
)

var pprofileotlp = &Package{
	info: &PackageInfo{
		name: "pprofileotlp",
		path: filepath.Join("pprofile", "pprofileotlp"),
		imports: []string{
			`otlpcollectorprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1development"`,
		},
		testImports: []string{
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
		},
	},
	structs: []baseStruct{
		exportProfilesPartialSuccess,
	},
}

var exportProfilesPartialSuccess = &messageStruct{
	structName:     "ExportPartialSuccess",
	description:    "// ExportPartialSuccess represents the details of a partially successful export request.",
	originFullName: "otlpcollectorprofile.ExportProfilesPartialSuccess",
	fields: []Field{
		&PrimitiveField{
			fieldName: "RejectedProfiles",
			protoID:   1,
			protoType: ProtoTypeInt64,
		},
		&PrimitiveField{
			fieldName: "ErrorMessage",
			protoID:   2,
			protoType: ProtoTypeString,
		},
	},
}
