// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"
import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var pprofile = &Package{
	info: &PackageInfo{
		name: "pprofile",
		path: "pprofile",
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			`"sync"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"strconv"`,
			`"testing"`,
			`"unsafe"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			`"google.golang.org/protobuf/proto"`,
			`gootlpcollectorprofiles "go.opentelemetry.io/proto/slim/otlp/collector/profiles/v1development"`,
			`gootlpcommon "go.opentelemetry.io/proto/slim/otlp/common/v1"`,
			`gootlpprofiles "go.opentelemetry.io/proto/slim/otlp/profiles/v1development"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		profiles,
		profilesData,
		resourceProfilesSlice,
		resourceProfiles,
		profilesDictionary,
		scopeProfilesSlice,
		scopeProfiles,
		profilesSlice,
		profile,
		valueTypeSlice,
		valueType,
		sampleSlice,
		sample,
		mappingSlice,
		mapping,
		locationSlice,
		location,
		lineSlice,
		line,
		functionSlice,
		function,
		keyValueAndUnitSlice,
		keyValueAndUnit,
		linkSlice,
		link,
		stackSlice,
		stack,
	},
}

var profiles = &messageStruct{
	structName:    "Profiles",
	description:   "// Profiles is the top-level struct that is propagated through the profiles pipeline.\n// Use NewProfiles to create new instance, zero-initialized instance is not valid for use.",
	protoName:     "ExportProfilesServiceRequest",
	upstreamProto: "gootlpcollectorprofiles.ExportProfilesServiceRequest",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceProfiles",
			protoID:     1,
			protoType:   proto.TypeMessage,
			returnSlice: resourceProfilesSlice,
		},
		&MessageField{
			fieldName:     "Dictionary",
			protoID:       2,
			returnMessage: profilesDictionary,
		},
	},
	hasWrapper: true,
}

var profilesData = &messageStruct{
	structName:    "ProfilesData",
	description:   "// ProfilesData represents the profiles data that can be stored in persistent storage,\n// OR can be embedded by other protocols that transfer OTLP profiles data but do not\n// implement the OTLP protocol.",
	protoName:     "ProfilesData",
	upstreamProto: "gootlpprofiles.ProfilesData",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceProfiles",
			protoID:     1,
			protoType:   proto.TypeMessage,
			returnSlice: resourceProfilesSlice,
		},
		&MessageField{
			fieldName:     "Dictionary",
			protoID:       2,
			returnMessage: profilesDictionary,
		},
	},
	hasWrapper: true,
}

var resourceProfilesSlice = &messageSlice{
	structName:      "ResourceProfilesSlice",
	elementNullable: true,
	element:         resourceProfiles,
}

var resourceProfiles = &messageStruct{
	structName:    "ResourceProfiles",
	description:   "// ResourceProfiles is a collection of profiles from a Resource.",
	protoName:     "ResourceProfiles",
	upstreamProto: "gootlpprofiles.ResourceProfiles",
	fields: []Field{
		&MessageField{
			fieldName:     "Resource",
			protoID:       1,
			returnMessage: resource,
		},
		&SliceField{
			fieldName:   "ScopeProfiles",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: scopeProfilesSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: proto.TypeString,
		},
	},
}

var profilesDictionary = &messageStruct{
	structName:    "ProfilesDictionary",
	description:   "// ProfilesDictionary is the reference table containing all data shared by profiles across the message being sent.",
	protoName:     "ProfilesDictionary",
	upstreamProto: "gootlpprofiles.ProfilesDictionary",
	fields: []Field{
		&SliceField{
			fieldName:   "MappingTable",
			protoID:     1,
			protoType:   proto.TypeMessage,
			returnSlice: mappingSlice,
		},
		&SliceField{
			fieldName:   "LocationTable",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: locationSlice,
		},
		&SliceField{
			fieldName:   "FunctionTable",
			protoID:     3,
			protoType:   proto.TypeMessage,
			returnSlice: functionSlice,
		},
		&SliceField{
			fieldName:   "LinkTable",
			protoID:     4,
			protoType:   proto.TypeMessage,
			returnSlice: linkSlice,
		},
		&SliceField{
			fieldName:   "StringTable",
			protoID:     5,
			protoType:   proto.TypeString,
			returnSlice: stringSlice,
		},
		&SliceField{
			fieldName:   "AttributeTable",
			protoID:     6,
			protoType:   proto.TypeMessage,
			returnSlice: keyValueAndUnitSlice,
		},
		&SliceField{
			fieldName:   "StackTable",
			protoID:     7,
			protoType:   proto.TypeMessage,
			returnSlice: stackSlice,
		},
	},
}

var scopeProfilesSlice = &messageSlice{
	structName:      "ScopeProfilesSlice",
	elementNullable: true,
	element:         scopeProfiles,
}

var scopeProfiles = &messageStruct{
	structName:    "ScopeProfiles",
	description:   "// ScopeProfiles is a collection of profiles from a LibraryInstrumentation.",
	protoName:     "ScopeProfiles",
	upstreamProto: "gootlpprofiles.ScopeProfiles",
	fields: []Field{
		&MessageField{
			fieldName:     "Scope",
			protoID:       1,
			returnMessage: scope,
		},
		&SliceField{
			fieldName:   "Profiles",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: profilesSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: proto.TypeString,
		},
	},
}

var profilesSlice = &messageSlice{
	structName:      "ProfilesSlice",
	elementNullable: true,
	element:         profile,
}

var profile = &messageStruct{
	structName:    "Profile",
	description:   "// Profile are an implementation of the pprofextended data model.\n",
	protoName:     "Profile",
	upstreamProto: "gootlpprofiles.Profile",
	fields: []Field{
		&MessageField{
			fieldName:     "SampleType",
			protoID:       1,
			returnMessage: valueType,
		},
		&SliceField{
			fieldName:   "Samples",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: sampleSlice,
		},
		&TypedField{
			fieldName:       "Time",
			originFieldName: "TimeUnixNano",
			protoID:         3,
			returnType:      timestampType,
		},
		&PrimitiveField{
			fieldName: "DurationNano",
			protoID:   4,
			protoType: proto.TypeUint64,
		},
		&MessageField{
			fieldName:     "PeriodType",
			protoID:       5,
			returnMessage: valueType,
		},
		&PrimitiveField{
			fieldName: "Period",
			protoID:   6,
			protoType: proto.TypeInt64,
		},
		&TypedField{
			fieldName:       "ProfileID",
			originFieldName: "ProfileId",
			protoID:         7,
			returnType: &TypedType{
				structName:  "ProfileID",
				protoType:   proto.TypeMessage,
				messageName: "ProfileID",
				defaultVal:  "ProfileID([16]byte{})",
				testVal:     "ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})",
			},
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   8,
			protoType: proto.TypeUint32,
		},
		&PrimitiveField{
			fieldName: "OriginalPayloadFormat",
			protoID:   9,
			protoType: proto.TypeString,
		},
		&SliceField{
			fieldName:   "OriginalPayload",
			protoID:     10,
			protoType:   proto.TypeBytes,
			returnSlice: byteSlice,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     11,
			protoType:   proto.TypeInt32,
			returnSlice: int32Slice,
		},
	},
}

var keyValueAndUnitSlice = &messageSlice{
	structName:      "KeyValueAndUnitSlice",
	elementNullable: true,
	element:         keyValueAndUnit,
}

var keyValueAndUnit = &messageStruct{
	structName: "KeyValueAndUnit",
	description: `// KeyValueAndUnit represents a custom 'dictionary native'
	// style of encoding attributes which is more convenient
	// for profiles than opentelemetry.proto.common.v1.KeyValue.`,
	protoName:     "KeyValueAndUnit",
	upstreamProto: "gootlpprofiles.KeyValueAndUnit",
	fields: []Field{
		&PrimitiveField{
			fieldName: "KeyStrindex",
			protoID:   1,
			protoType: proto.TypeInt32,
		},
		&MessageField{
			fieldName:     "Value",
			protoID:       2,
			returnMessage: anyValueStruct,
		},
		&PrimitiveField{
			fieldName: "UnitStrindex",
			protoID:   3,
			protoType: proto.TypeInt32,
		},
	},
}

var linkSlice = &messageSlice{
	structName:      "LinkSlice",
	elementNullable: true,
	element:         link,
}

var link = &messageStruct{
	structName:    "Link",
	description:   "// Link represents a pointer from a profile Sample to a trace Span.",
	protoName:     "Link",
	upstreamProto: "gootlpprofiles.Link",
	fields: []Field{
		&TypedField{
			fieldName:       "TraceID",
			originFieldName: "TraceId",
			protoID:         1,
			returnType:      traceIDType,
		},
		&TypedField{
			fieldName:       "SpanID",
			originFieldName: "SpanId",
			protoID:         2,
			returnType:      spanIDType,
		},
	},
}

var valueTypeSlice = &messageSlice{
	structName:      "ValueTypeSlice",
	elementNullable: true,
	element:         valueType,
}

var valueType = &messageStruct{
	structName:    "ValueType",
	description:   "// ValueType describes the type and units of a value.",
	protoName:     "ValueType",
	upstreamProto: "gootlpprofiles.ValueType",
	fields: []Field{
		&PrimitiveField{
			fieldName: "TypeStrindex",
			protoID:   1,
			protoType: proto.TypeInt32,
		},
		&PrimitiveField{
			fieldName: "UnitStrindex",
			protoID:   2,
			protoType: proto.TypeInt32,
		},
	},
}

var sampleSlice = &messageSlice{
	structName:      "SampleSlice",
	elementNullable: true,
	element:         sample,
}

var sample = &messageStruct{
	structName:    "Sample",
	description:   "// Sample represents each record value encountered within a profiled program.",
	protoName:     "Sample",
	upstreamProto: "gootlpprofiles.Sample",
	fields: []Field{
		&PrimitiveField{
			fieldName: "StackIndex",
			protoID:   1,
			protoType: proto.TypeInt32,
		},
		&SliceField{
			fieldName:   "Values",
			protoID:     2,
			protoType:   proto.TypeInt64,
			returnSlice: int64Slice,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     3,
			protoType:   proto.TypeInt32,
			returnSlice: int32Slice,
		},
		&PrimitiveField{
			fieldName: "LinkIndex",
			protoID:   4,
			protoType: proto.TypeInt32,
		},
		&SliceField{
			fieldName:   "TimestampsUnixNano",
			protoID:     5,
			protoType:   proto.TypeFixed64,
			returnSlice: uInt64Slice,
		},
	},
}

var mappingSlice = &messageSlice{
	structName:      "MappingSlice",
	elementNullable: true,
	element:         mapping,
}

var mapping = &messageStruct{
	structName:    "Mapping",
	description:   "// Mapping describes the mapping of a binary in memory, including its address range, file offset, and metadata like build ID",
	protoName:     "Mapping",
	upstreamProto: "gootlpprofiles.Mapping",
	fields: []Field{
		&PrimitiveField{
			fieldName: "MemoryStart",
			protoID:   1,
			protoType: proto.TypeUint64,
		},
		&PrimitiveField{
			fieldName: "MemoryLimit",
			protoID:   2,
			protoType: proto.TypeUint64,
		},
		&PrimitiveField{
			fieldName: "FileOffset",
			protoID:   3,
			protoType: proto.TypeUint64,
		},
		&PrimitiveField{
			fieldName: "FilenameStrindex",
			protoID:   4,
			protoType: proto.TypeInt32,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     5,
			protoType:   proto.TypeInt32,
			returnSlice: int32Slice,
		},
	},
}

var locationSlice = &messageSlice{
	structName:      "LocationSlice",
	elementNullable: true,
	element:         location,
}

var location = &messageStruct{
	structName:    "Location",
	description:   "// Location describes function and line table debug information.",
	protoName:     "Location",
	upstreamProto: "gootlpprofiles.Location",
	fields: []Field{
		&PrimitiveField{
			fieldName: "MappingIndex",
			protoID:   1,
			protoType: proto.TypeInt32,
		},
		&PrimitiveField{
			fieldName: "Address",
			protoID:   2,
			protoType: proto.TypeUint64,
		},
		&SliceField{
			fieldName:   "Lines",
			protoID:     3,
			protoType:   proto.TypeMessage,
			returnSlice: lineSlice,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     4,
			protoType:   proto.TypeInt32,
			returnSlice: int32Slice,
		},
	},
}

var lineSlice = &messageSlice{
	structName:      "LineSlice",
	elementNullable: true,
	element:         line,
}

var line = &messageStruct{
	structName:    "Line",
	description:   "// Line details a specific line in a source code, linked to a function.",
	protoName:     "Line",
	upstreamProto: "gootlpprofiles.Line",
	fields: []Field{
		&PrimitiveField{
			fieldName: "FunctionIndex",
			protoID:   1,
			protoType: proto.TypeInt32,
		},
		&PrimitiveField{
			fieldName: "Line",
			protoID:   2,
			protoType: proto.TypeInt64,
		},
		&PrimitiveField{
			fieldName: "Column",
			protoID:   3,
			protoType: proto.TypeInt64,
		},
	},
}

var functionSlice = &messageSlice{
	structName:      "FunctionSlice",
	elementNullable: true,
	element:         function,
}

var function = &messageStruct{
	structName:    "Function",
	description:   "// Function describes a function, including its human-readable name, system name, source file, and starting line number in the source.",
	protoName:     "Function",
	upstreamProto: "gootlpprofiles.Function",
	fields: []Field{
		&PrimitiveField{
			fieldName: "NameStrindex",
			protoID:   1,
			protoType: proto.TypeInt32,
		},
		&PrimitiveField{
			fieldName: "SystemNameStrindex",
			protoID:   2,
			protoType: proto.TypeInt32,
		},
		&PrimitiveField{
			fieldName: "FilenameStrindex",
			protoID:   3,
			protoType: proto.TypeInt32,
		},
		&PrimitiveField{
			fieldName: "StartLine",
			protoID:   4,
			protoType: proto.TypeInt64,
		},
	},
}

var stackSlice = &messageSlice{
	structName:      "StackSlice",
	elementNullable: true,
	element:         stack,
}

var stack = &messageStruct{
	structName:    "Stack",
	description:   "// Stack represents a stack trace as a list of locations.\n",
	protoName:     "Stack",
	upstreamProto: "gootlpprofiles.Stack",
	fields: []Field{
		&SliceField{
			fieldName:   "LocationIndices",
			protoID:     1,
			protoType:   proto.TypeInt32,
			returnSlice: int32Slice,
		},
	},
}
