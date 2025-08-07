// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

import (
	"path/filepath"
)

var pmetricotlp = &Package{
	info: &PackageInfo{
		name: "pmetricotlp",
		path: filepath.Join("pmetric", "pmetricotlp"),
		imports: []string{
			`otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"`,
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
		exportMetricsPartialSuccess,
	},
}

var exportMetricsPartialSuccess = &messageStruct{
	structName:     "ExportPartialSuccess",
	description:    "// ExportPartialSuccess represents the details of a partially successful export request.",
	originFullName: "otlpcollectormetrics.ExportMetricsPartialSuccess",
	fields: []Field{
		&PrimitiveField{
			fieldName: "RejectedDataPoints",
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
