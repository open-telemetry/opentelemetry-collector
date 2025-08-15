// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

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
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`otlpcollectorprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1development"`,
			`otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"`,
			`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
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
			`otlpcollectorprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1development"`,
			`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		profiles,
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
		attributeTableSlice,
		attribute,
		attributeUnitSlice,
		attributeUnit,
		linkSlice,
		link,
	},
}

var profiles = &messageStruct{
	structName:     "Profiles",
	description:    "// Profiles is the top-level struct that is propagated through the profiles pipeline.\n// Use NewProfiles to create new instance, zero-initialized instance is not valid for use.",
	originFullName: "otlpcollectorprofiles.ExportProfilesServiceRequest",
	fields: []Field{
		&SliceField{
			fieldName:   "ResourceProfiles",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: resourceProfilesSlice,
		},
		&MessageField{
			fieldName:           "ProfilesDictionary",
			fieldOriginFullName: "Dictionary",
			protoID:             2,
			returnMessage:       profilesDictionary,
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
	structName:     "ResourceProfiles",
	description:    "// ResourceProfiles is a collection of profiles from a Resource.",
	originFullName: "otlpprofiles.ResourceProfiles",
	fields: []Field{
		&MessageField{
			fieldName:     "Resource",
			protoID:       1,
			returnMessage: resource,
		},
		&SliceField{
			fieldName:   "ScopeProfiles",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: scopeProfilesSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: ProtoTypeString,
		},
	},
}

var profilesDictionary = &messageStruct{
	structName:     "ProfilesDictionary",
	description:    "// ProfilesDictionary is the reference table containing all data shared by profiles across the message being sent.",
	originFullName: "otlpprofiles.ProfilesDictionary",
	fields: []Field{
		&SliceField{
			fieldName:   "MappingTable",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: mappingSlice,
		},
		&SliceField{
			fieldName:   "LocationTable",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: locationSlice,
		},
		&SliceField{
			fieldName:   "FunctionTable",
			protoID:     3,
			protoType:   ProtoTypeMessage,
			returnSlice: functionSlice,
		},
		&SliceField{
			fieldName:   "LinkTable",
			protoID:     4,
			protoType:   ProtoTypeMessage,
			returnSlice: linkSlice,
		},
		&SliceField{
			fieldName:   "StringTable",
			protoID:     5,
			protoType:   ProtoTypeString,
			returnSlice: stringSlice,
		},
		&SliceField{
			fieldName:   "AttributeTable",
			protoID:     6,
			protoType:   ProtoTypeMessage,
			returnSlice: attributeTableSlice,
		},
		&SliceField{
			fieldName:   "AttributeUnits",
			protoID:     7,
			protoType:   ProtoTypeMessage,
			returnSlice: attributeUnitSlice,
		},
	},
}

var scopeProfilesSlice = &messageSlice{
	structName:      "ScopeProfilesSlice",
	elementNullable: true,
	element:         scopeProfiles,
}

var scopeProfiles = &messageStruct{
	structName:     "ScopeProfiles",
	description:    "// ScopeProfiles is a collection of profiles from a LibraryInstrumentation.",
	originFullName: "otlpprofiles.ScopeProfiles",
	fields: []Field{
		&MessageField{
			fieldName:     "Scope",
			protoID:       1,
			returnMessage: scope,
		},
		&SliceField{
			fieldName:   "Profiles",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: profilesSlice,
		},
		&PrimitiveField{
			fieldName: "SchemaUrl",
			protoID:   3,
			protoType: ProtoTypeString,
		},
	},
}

var profilesSlice = &messageSlice{
	structName:      "ProfilesSlice",
	elementNullable: true,
	element:         profile,
}

var profile = &messageStruct{
	structName:     "Profile",
	description:    "// Profile are an implementation of the pprofextended data model.\n",
	originFullName: "otlpprofiles.Profile",
	fields: []Field{
		&SliceField{
			fieldName:   "SampleType",
			protoID:     1,
			protoType:   ProtoTypeMessage,
			returnSlice: valueTypeSlice,
		},
		&SliceField{
			fieldName:   "Sample",
			protoID:     2,
			protoType:   ProtoTypeMessage,
			returnSlice: sampleSlice,
		},
		&SliceField{
			fieldName:   "LocationIndices",
			protoID:     3,
			protoType:   ProtoTypeInt32,
			returnSlice: int32Slice,
		},
		&TypedField{
			fieldName:       "Time",
			originFieldName: "TimeNanos",
			protoID:         4,
			returnType: &TypedType{
				structName:  "Timestamp",
				packageName: "pcommon",
				protoType:   ProtoTypeInt64,
				defaultVal:  "0",
				testVal:     "1234567890",
			},
		},
		&TypedField{
			fieldName:       "Duration",
			originFieldName: "DurationNanos",
			protoID:         5,
			returnType: &TypedType{
				structName:  "Timestamp",
				packageName: "pcommon",
				protoType:   ProtoTypeInt64,
				defaultVal:  "0",
				testVal:     "1234567890",
			},
		},
		&MessageField{
			fieldName:     "PeriodType",
			protoID:       6,
			returnMessage: valueType,
		},
		&PrimitiveField{
			fieldName: "Period",
			protoID:   7,
			protoType: ProtoTypeInt64,
		},
		&SliceField{
			fieldName:   "CommentStrindices",
			protoID:     8,
			protoType:   ProtoTypeInt32,
			returnSlice: int32Slice,
		},
		&PrimitiveField{
			fieldName: "DefaultSampleTypeIndex",
			protoID:   9,
			protoType: ProtoTypeInt32,
		},
		&TypedField{
			fieldName:       "ProfileID",
			originFieldName: "ProfileId",
			protoID:         10,
			returnType: &TypedType{
				structName:  "ProfileID",
				protoType:   ProtoTypeMessage,
				messageName: "data.ProfileID",
				defaultVal:  "data.ProfileID([16]byte{})",
				testVal:     "data.ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})",
			},
		},
		&PrimitiveField{
			fieldName: "DroppedAttributesCount",
			protoID:   11,
			protoType: ProtoTypeUint32,
		},
		&PrimitiveField{
			fieldName: "OriginalPayloadFormat",
			protoID:   12,
			protoType: ProtoTypeString,
		},
		&SliceField{
			fieldName:   "OriginalPayload",
			protoID:     13,
			protoType:   ProtoTypeBytes,
			returnSlice: byteSlice,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     14,
			protoType:   ProtoTypeInt32,
			returnSlice: int32Slice,
		},
	},
}

var attributeUnitSlice = &messageSlice{
	structName:      "AttributeUnitSlice",
	elementNullable: true,
	element:         attributeUnit,
}

var attributeUnit = &messageStruct{
	structName:     "AttributeUnit",
	description:    "// AttributeUnit Represents a mapping between Attribute Keys and Units.",
	originFullName: "otlpprofiles.AttributeUnit",
	fields: []Field{
		&PrimitiveField{
			fieldName: "AttributeKeyStrindex",
			protoID:   1,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "UnitStrindex",
			protoID:   2,
			protoType: ProtoTypeInt32,
		},
	},
}

var linkSlice = &messageSlice{
	structName:      "LinkSlice",
	elementNullable: true,
	element:         link,
}

var link = &messageStruct{
	structName:     "Link",
	description:    "// Link represents a pointer from a profile Sample to a trace Span.",
	originFullName: "otlpprofiles.Link",
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
	structName:     "ValueType",
	description:    "// ValueType describes the type and units of a value, with an optional aggregation temporality.",
	originFullName: "otlpprofiles.ValueType",
	fields: []Field{
		&PrimitiveField{
			fieldName: "TypeStrindex",
			protoID:   1,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "UnitStrindex",
			protoID:   2,
			protoType: ProtoTypeInt32,
		},
		&TypedField{
			fieldName: "AggregationTemporality",
			protoID:   3,
			returnType: &TypedType{
				structName:  "AggregationTemporality",
				protoType:   ProtoTypeEnum,
				messageName: "otlpprofiles.AggregationTemporality",
				defaultVal:  "otlpprofiles.AggregationTemporality(0)",
				testVal:     "otlpprofiles.AggregationTemporality(1)",
			},
		},
	},
}

var sampleSlice = &messageSlice{
	structName:      "SampleSlice",
	elementNullable: true,
	element:         sample,
}

var sample = &messageStruct{
	structName:     "Sample",
	description:    "// Sample represents each record value encountered within a profiled program.",
	originFullName: "otlpprofiles.Sample",
	fields: []Field{
		&PrimitiveField{
			fieldName: "LocationsStartIndex",
			protoID:   1,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "LocationsLength",
			protoID:   2,
			protoType: ProtoTypeInt32,
		},
		&SliceField{
			fieldName:   "Value",
			protoID:     3,
			protoType:   ProtoTypeInt64,
			returnSlice: int64Slice,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     4,
			protoType:   ProtoTypeInt32,
			returnSlice: int32Slice,
		},
		&OptionalPrimitiveField{
			fieldName: "LinkIndex",
			protoID:   5,
			protoType: ProtoTypeInt32,
		},
		&SliceField{
			fieldName:   "TimestampsUnixNano",
			protoID:     6,
			protoType:   ProtoTypeUint64,
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
	structName:     "Mapping",
	description:    "// Mapping describes the mapping of a binary in memory, including its address range, file offset, and metadata like build ID",
	originFullName: "otlpprofiles.Mapping",
	fields: []Field{
		&PrimitiveField{
			fieldName: "MemoryStart",
			protoID:   1,
			protoType: ProtoTypeUint64,
		},
		&PrimitiveField{
			fieldName: "MemoryLimit",
			protoID:   2,
			protoType: ProtoTypeUint64,
		},
		&PrimitiveField{
			fieldName: "FileOffset",
			protoID:   3,
			protoType: ProtoTypeUint64,
		},
		&PrimitiveField{
			fieldName: "FilenameStrindex",
			protoID:   4,
			protoType: ProtoTypeInt32,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     5,
			protoType:   ProtoTypeInt32,
			returnSlice: int32Slice,
		},
		&PrimitiveField{
			fieldName: "HasFunctions",
			protoID:   6,
			protoType: ProtoTypeBool,
		},
		&PrimitiveField{
			fieldName: "HasFilenames",
			protoID:   7,
			protoType: ProtoTypeBool,
		},
		&PrimitiveField{
			fieldName: "HasLineNumbers",
			protoID:   8,
			protoType: ProtoTypeBool,
		},
		&PrimitiveField{
			fieldName: "HasInlineFrames",
			protoID:   9,
			protoType: ProtoTypeBool,
		},
	},
}

var locationSlice = &messageSlice{
	structName:      "LocationSlice",
	elementNullable: true,
	element:         location,
}

var location = &messageStruct{
	structName:     "Location",
	description:    "// Location describes function and line table debug information.",
	originFullName: "otlpprofiles.Location",
	fields: []Field{
		&OptionalPrimitiveField{
			fieldName: "MappingIndex",
			protoID:   1,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "Address",
			protoID:   2,
			protoType: ProtoTypeUint64,
		},
		&SliceField{
			fieldName:   "Line",
			protoID:     3,
			protoType:   ProtoTypeMessage,
			returnSlice: lineSlice,
		},
		&PrimitiveField{
			fieldName: "IsFolded",
			protoID:   4,
			protoType: ProtoTypeBool,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			protoID:     5,
			protoType:   ProtoTypeInt32,
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
	structName:     "Line",
	description:    "// Line details a specific line in a source code, linked to a function.",
	originFullName: "otlpprofiles.Line",
	fields: []Field{
		&PrimitiveField{
			fieldName: "FunctionIndex",
			protoID:   1,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "Line",
			protoID:   2,
			protoType: ProtoTypeInt64,
		},
		&PrimitiveField{
			fieldName: "Column",
			protoID:   3,
			protoType: ProtoTypeInt64,
		},
	},
}

var functionSlice = &messageSlice{
	structName:      "FunctionSlice",
	elementNullable: true,
	element:         function,
}

var function = &messageStruct{
	structName:     "Function",
	description:    "// Function describes a function, including its human-readable name, system name, source file, and starting line number in the source.",
	originFullName: "otlpprofiles.Function",
	fields: []Field{
		&PrimitiveField{
			fieldName: "NameStrindex",
			protoID:   1,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "SystemNameStrindex",
			protoID:   2,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "FilenameStrindex",
			protoID:   3,
			protoType: ProtoTypeInt32,
		},
		&PrimitiveField{
			fieldName: "StartLine",
			protoID:   4,
			protoType: ProtoTypeInt64,
		},
	},
}

var attributeTableSlice = &messageSlice{
	structName:      "AttributeTableSlice",
	elementNullable: false,
	element:         attribute,
}

var attribute = &messageStruct{
	structName:     "Attribute",
	description:    "// Attribute describes an attribute stored in a profile's attribute table.",
	originFullName: "otlpcommon.KeyValue",
	fields: []Field{
		&PrimitiveField{
			fieldName: "Key",
			protoID:   1,
			protoType: ProtoTypeString,
		},
		&MessageField{
			fieldName:     "Value",
			protoID:       2,
			returnMessage: anyValue,
		},
	},
}
