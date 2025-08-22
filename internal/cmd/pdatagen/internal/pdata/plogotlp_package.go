// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
import (
	"path/filepath"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var plogotlp = &Package{
	info: &PackageInfo{
		name: "plogotlp",
		path: filepath.Join("plog", "plogotlp"),
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			`"sync"`,
			``,
			`otlpcollectorlogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"`,
		},
		testImports: []string{
			`"strconv"`,
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			`"google.golang.org/protobuf/proto"`,
			`gootlpcollectorlogs "go.opentelemetry.io/proto/slim/otlp/collector/logs/v1"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`otlpcollectorlogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"`,
		},
	},
	structs: []baseStruct{
		exportLogsResponse,
		exportLogsPartialSuccess,
	},
}

var exportLogsResponse = &messageStruct{
	structName:     "ExportResponse",
	description:    "// ExportResponse represents the response for gRPC/HTTP client/server.",
	originFullName: "otlpcollectorlogs.ExportLogsServiceResponse",
	fields: []Field{
		&MessageField{
			fieldName:     "PartialSuccess",
			protoID:       1,
			returnMessage: exportLogsPartialSuccess,
		},
	},
}

var exportLogsPartialSuccess = &messageStruct{
	structName:     "ExportPartialSuccess",
	description:    "// ExportPartialSuccess represents the details of a partially successful export request.",
	originFullName: "otlpcollectorlogs.ExportLogsPartialSuccess",
	fields: []Field{
		&PrimitiveField{
			fieldName: "RejectedLogRecords",
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
