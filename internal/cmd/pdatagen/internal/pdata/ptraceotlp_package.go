// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"path/filepath"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var ptraceotlp = &Package{
	info: &PackageInfo{
		name: "ptraceotlp",
		path: filepath.Join("ptrace", "ptraceotlp"),
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			`"sync"`,
			``,
			`otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"`,
		},
		testImports: []string{
			`"strconv"`,
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			`"google.golang.org/protobuf/proto"`,
			`gootlpcollectortrace "go.opentelemetry.io/proto/slim/otlp/collector/trace/v1"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"`,
		},
	},
	structs: []baseStruct{
		exportTraceResponse,
		exportTracePartialSuccess,
	},
}

var exportTraceResponse = &messageStruct{
	structName:     "ExportResponse",
	description:    "// ExportResponse represents the response for gRPC/HTTP client/server.",
	originFullName: "otlpcollectortrace.ExportTraceServiceResponse",
	fields: []Field{
		&MessageField{
			fieldName:     "PartialSuccess",
			protoID:       1,
			returnMessage: exportTracePartialSuccess,
		},
	},
}

var exportTracePartialSuccess = &messageStruct{
	structName:     "ExportPartialSuccess",
	description:    "// ExportPartialSuccess represents the details of a partially successful export request.",
	originFullName: "otlpcollectortrace.ExportTracePartialSuccess",
	fields: []Field{
		&PrimitiveField{
			fieldName: "RejectedSpans",
			protoID:   1,
			protoType: proto.TypeInt64,
		},
		&PrimitiveField{
			fieldName: "ErrorMessage",
			protoID:   2,
			protoType: proto.TypeString,
		},
	},
}
