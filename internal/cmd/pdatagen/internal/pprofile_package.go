// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

var pprofile = &Package{
	info: &PackageInfo{
		name: "pprofile",
		path: "pprofile",
		imports: []string{
			`"iter"`,
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"testing"`,
			`"unsafe"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"github.com/stretchr/testify/require"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
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

var resourceProfilesSlice = &sliceOfPtrs{
	structName: "ResourceProfilesSlice",
	element:    resourceProfiles,
}

var resourceProfiles = &messageStruct{
	structName:     "ResourceProfiles",
	description:    "// ResourceProfiles is a collection of profiles from a Resource.",
	originFullName: "otlpprofiles.ResourceProfiles",
	fields: []Field{
		resourceField,
		schemaURLField,
		&SliceField{
			fieldName:   "ScopeProfiles",
			returnSlice: scopeProfilesSlice,
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
			returnSlice: mappingSlice,
		},
		&SliceField{
			fieldName:   "LocationTable",
			returnSlice: locationSlice,
		},
		&SliceField{
			fieldName:   "FunctionTable",
			returnSlice: functionSlice,
		},
		&SliceField{
			fieldName:   "LinkTable",
			returnSlice: linkSlice,
		},
		&SliceField{
			fieldName:   "StringTable",
			returnSlice: stringSlice,
		},
		&SliceField{
			fieldName:   "AttributeTable",
			returnSlice: attributeTableSlice,
		},
		&SliceField{
			fieldName:   "AttributeUnits",
			returnSlice: attributeUnitSlice,
		},
	},
}

var scopeProfilesSlice = &sliceOfPtrs{
	structName: "ScopeProfilesSlice",
	element:    scopeProfiles,
}

var scopeProfiles = &messageStruct{
	structName:     "ScopeProfiles",
	description:    "// ScopeProfiles is a collection of profiles from a LibraryInstrumentation.",
	originFullName: "otlpprofiles.ScopeProfiles",
	fields: []Field{
		scopeField,
		schemaURLField,
		&SliceField{
			fieldName:   "Profiles",
			returnSlice: profilesSlice,
		},
	},
}

var profilesSlice = &sliceOfPtrs{
	structName: "ProfilesSlice",
	element:    profile,
}

var profile = &messageStruct{
	structName:     "Profile",
	description:    "// Profile are an implementation of the pprofextended data model.\n",
	originFullName: "otlpprofiles.Profile",
	fields: []Field{
		&SliceField{
			fieldName:   "SampleType",
			returnSlice: valueTypeSlice,
		},
		&SliceField{
			fieldName:   "Sample",
			returnSlice: sampleSlice,
		},
		&SliceField{
			fieldName:   "LocationIndices",
			returnSlice: int32Slice,
		},
		&TypedField{
			fieldName:       "Time",
			originFieldName: "TimeNanos",
			returnType: &TypedType{
				structName:  "Timestamp",
				packageName: "pcommon",
				rawType:     "int64",
				defaultVal:  "0",
				testVal:     "1234567890",
			},
		},
		&TypedField{
			fieldName:       "Duration",
			originFieldName: "DurationNanos",
			returnType: &TypedType{
				structName:  "Timestamp",
				packageName: "pcommon",
				rawType:     "int64",
				defaultVal:  "0",
				testVal:     "1234567890",
			},
		},
		&MessageField{
			fieldName:     "PeriodType",
			returnMessage: valueType,
		},
		&PrimitiveField{
			fieldName:  "Period",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&SliceField{
			fieldName:   "CommentStrindices",
			returnSlice: int32Slice,
		},
		&PrimitiveField{
			fieldName:  "DefaultSampleTypeIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&TypedField{
			fieldName:       "ProfileID",
			originFieldName: "ProfileId",
			returnType: &TypedType{
				structName: "ProfileID",
				rawType:    "data.ProfileID",
				isType:     true,
				defaultVal: "data.ProfileID([16]byte{})",
				testVal:    "data.ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})",
			},
		},
		droppedAttributesCount,
		&PrimitiveField{
			fieldName:  "OriginalPayloadFormat",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"original payload"`,
		},
		&SliceField{
			fieldName:   "OriginalPayload",
			returnSlice: byteSlice,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
	},
}

var attributeUnitSlice = &sliceOfPtrs{
	structName: "AttributeUnitSlice",
	element:    attributeUnit,
}

var attributeUnit = &messageStruct{
	structName:     "AttributeUnit",
	description:    "// AttributeUnit Represents a mapping between Attribute Keys and Units.",
	originFullName: "otlpprofiles.AttributeUnit",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "AttributeKeyStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "UnitStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
	},
}

var linkSlice = &sliceOfPtrs{
	structName: "LinkSlice",
	element:    link,
}

var link = &messageStruct{
	structName:     "Link",
	description:    "// Link represents a pointer from a profile Sample to a trace Span.",
	originFullName: "otlpprofiles.Link",
	fields: []Field{
		traceIDField,
		spanIDField,
	},
}

var valueTypeSlice = &sliceOfPtrs{
	structName: "ValueTypeSlice",
	element:    valueType,
}

var valueType = &messageStruct{
	structName:     "ValueType",
	description:    "// ValueType describes the type and units of a value, with an optional aggregation temporality.",
	originFullName: "otlpprofiles.ValueType",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "TypeStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "UnitStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&TypedField{
			fieldName: "AggregationTemporality",
			returnType: &TypedType{
				structName: "AggregationTemporality",
				rawType:    "otlpprofiles.AggregationTemporality",
				isEnum:     true,
				defaultVal: "otlpprofiles.AggregationTemporality(0)",
				testVal:    "otlpprofiles.AggregationTemporality(1)",
			},
		},
	},
}

var sampleSlice = &sliceOfPtrs{
	structName: "SampleSlice",
	element:    sample,
}

var sample = &messageStruct{
	structName:     "Sample",
	description:    "// Sample represents each record value encountered within a profiled program.",
	originFullName: "otlpprofiles.Sample",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "LocationsStartIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "LocationsLength",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&SliceField{
			fieldName:   "Value",
			returnSlice: int64Slice,
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
		&OptionalPrimitiveField{
			fieldName:  "LinkIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&SliceField{
			fieldName:   "TimestampsUnixNano",
			returnSlice: uInt64Slice,
		},
	},
}

var mappingSlice = &sliceOfPtrs{
	structName: "MappingSlice",
	element:    mapping,
}

var mapping = &messageStruct{
	structName:     "Mapping",
	description:    "// Mapping describes the mapping of a binary in memory, including its address range, file offset, and metadata like build ID",
	originFullName: "otlpprofiles.Mapping",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "MemoryStart",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&PrimitiveField{
			fieldName:  "MemoryLimit",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&PrimitiveField{
			fieldName:  "FileOffset",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&PrimitiveField{
			fieldName:  "FilenameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
		&PrimitiveField{
			fieldName:  "HasFunctions",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&PrimitiveField{
			fieldName:  "HasFilenames",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&PrimitiveField{
			fieldName:  "HasLineNumbers",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&PrimitiveField{
			fieldName:  "HasInlineFrames",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
	},
}

var locationSlice = &sliceOfPtrs{
	structName: "LocationSlice",
	element:    location,
}

var location = &messageStruct{
	structName:     "Location",
	description:    "// Location describes function and line table debug information.",
	originFullName: "otlpprofiles.Location",
	fields: []Field{
		&OptionalPrimitiveField{
			fieldName:  "MappingIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "Address",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&SliceField{
			fieldName:   "Line",
			returnSlice: lineSlice,
		},
		&PrimitiveField{
			fieldName:  "IsFolded",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&SliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
	},
}

var lineSlice = &sliceOfPtrs{
	structName: "LineSlice",
	element:    line,
}

var line = &messageStruct{
	structName:     "Line",
	description:    "// Line details a specific line in a source code, linked to a function.",
	originFullName: "otlpprofiles.Line",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "FunctionIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "Line",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&PrimitiveField{
			fieldName:  "Column",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
	},
}

var functionSlice = &sliceOfPtrs{
	structName: "FunctionSlice",
	element:    function,
}

var function = &messageStruct{
	structName:     "Function",
	description:    "// Function describes a function, including its human-readable name, system name, source file, and starting line number in the source.",
	originFullName: "otlpprofiles.Function",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "NameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "SystemNameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "FilenameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&PrimitiveField{
			fieldName:  "StartLine",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
	},
}

var attributeTableSlice = &sliceOfValues{
	structName: "AttributeTableSlice",
	element:    attribute,
}

var attribute = &messageStruct{
	structName:     "Attribute",
	description:    "// Attribute describes an attribute stored in a profile's attribute table.",
	originFullName: "v1.KeyValue",
	fields: []Field{
		&PrimitiveField{
			fieldName:  "Key",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"key"`,
		},
		&MessageField{
			fieldName:     "Value",
			returnMessage: anyValue,
		},
	},
}
