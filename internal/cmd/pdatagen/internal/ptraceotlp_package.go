// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"path/filepath"
)

var ptraceotlp = &Package{
	info: &PackageInfo{
		name: "ptraceotlp",
		path: filepath.Join("ptrace", "ptraceotlp"),
		imports: []string{
			`otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"`,
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
		exportTracePartialSuccess,
	},
}

var exportTracePartialSuccess = &messageStruct{
	structName:     "ExportPartialSuccess",
	description:    "// ExportPartialSuccess represents the details of a partially successful export request.",
	originFullName: "otlpcollectortrace.ExportTracePartialSuccess",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "RejectedSpans",
			returnType: "int64",
			defaultVal: `int64(0)`,
			testVal:    `int64(13)`,
		},
		&PrimitiveField{
			fieldName:  "ErrorMessage",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"error message"`,
		},
	},
}
