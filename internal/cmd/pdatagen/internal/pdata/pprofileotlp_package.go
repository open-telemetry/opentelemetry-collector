// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
import (
	"path/filepath"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var pprofileotlp = &Package{
	info: &PackageInfo{
		name: "pprofileotlp",
		path: filepath.Join("pprofile", "pprofileotlp"),
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			`"sync"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
		},
		testImports: []string{
			`"strconv"`,
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			`"google.golang.org/protobuf/proto"`,
			`gootlpcollectorprofiles "go.opentelemetry.io/proto/slim/otlp/collector/profiles/v1development"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
		},
	},
	structs: []baseStruct{
		exportProfilesResponse,
		exportProfilesPartialSuccess,
	},
}

var exportProfilesResponse = &messageStruct{
	structName:    "ExportResponse",
	description:   "// ExportResponse represents the response for gRPC/HTTP client/server.",
	protoName:     "ExportProfilesServiceResponse",
	upstreamProto: "gootlpcollectorprofiles.ExportProfilesServiceResponse",
	fields: []Field{
		&MessageField{
			fieldName:     "PartialSuccess",
			protoID:       1,
			returnMessage: exportProfilesPartialSuccess,
		},
	},
}

var exportProfilesPartialSuccess = &messageStruct{
	structName:    "ExportPartialSuccess",
	description:   "// ExportPartialSuccess represents the details of a partially successful export request.",
	protoName:     "ExportProfilesPartialSuccess",
	upstreamProto: "gootlpcollectorprofiles.ExportProfilesPartialSuccess",
	fields: []Field{
		&PrimitiveField{
			fieldName: "RejectedProfiles",
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
