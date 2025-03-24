// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var pprofile = &Package{
	info: &PackageInfo{
		name: "pprofile",
		path: "pprofile",
		imports: []string{
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"testing"`,
			`"unsafe"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		resourceProfilesSlice,
		resourceProfiles,
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

var resourceProfiles = &messageValueStruct{
	structName:     "ResourceProfiles",
	description:    "// ResourceProfiles is a collection of profiles from a Resource.",
	originFullName: "otlpprofiles.ResourceProfiles",
	fields: []baseField{
		resourceField,
		schemaURLField,
		&sliceField{
			fieldName:   "ScopeProfiles",
			returnSlice: scopeProfilesSlice,
		},
	},
}

var scopeProfilesSlice = &sliceOfPtrs{
	structName: "ScopeProfilesSlice",
	element:    scopeProfiles,
}

var scopeProfiles = &messageValueStruct{
	structName:     "ScopeProfiles",
	description:    "// ScopeProfiles is a collection of profiles from a LibraryInstrumentation.",
	originFullName: "otlpprofiles.ScopeProfiles",
	fields: []baseField{
		scopeField,
		schemaURLField,
		&sliceField{
			fieldName:   "Profiles",
			returnSlice: profilesSlice,
		},
	},
}

var profilesSlice = &sliceOfPtrs{
	structName: "ProfilesSlice",
	element:    profile,
}

var profile = &messageValueStruct{
	structName:     "Profile",
	description:    "// Profile are an implementation of the pprofextended data model.\n",
	originFullName: "otlpprofiles.Profile",
	fields: []baseField{
		&sliceField{
			fieldName:   "SampleType",
			returnSlice: valueTypeSlice,
		},
		&sliceField{
			fieldName:   "Sample",
			returnSlice: sampleSlice,
		},
		&sliceField{
			fieldName:   "MappingTable",
			returnSlice: mappingSlice,
		},
		&sliceField{
			fieldName:   "LocationTable",
			returnSlice: locationSlice,
		},
		&sliceField{
			fieldName:   "LocationIndices",
			returnSlice: int32Slice,
		},
		&sliceField{
			fieldName:   "FunctionTable",
			returnSlice: functionSlice,
		},
		&sliceField{
			fieldName:   "AttributeTable",
			returnSlice: attributeTableSlice,
		},
		&sliceField{
			fieldName:   "AttributeUnits",
			returnSlice: attributeUnitSlice,
		},
		&sliceField{
			fieldName:   "LinkTable",
			returnSlice: linkSlice,
		},
		&sliceField{
			fieldName:   "StringTable",
			returnSlice: stringSlice,
		},
		&primitiveTypedField{
			fieldName:       "Time",
			originFieldName: "TimeNanos",
			returnType: &primitiveType{
				structName:  "Timestamp",
				packageName: "pcommon",
				rawType:     "int64",
				defaultVal:  "0",
				testVal:     "1234567890",
			},
		},
		&primitiveTypedField{
			fieldName:       "Duration",
			originFieldName: "DurationNanos",
			returnType: &primitiveType{
				structName:  "Timestamp",
				packageName: "pcommon",
				rawType:     "int64",
				defaultVal:  "0",
				testVal:     "1234567890",
			},
		},
		&primitiveTypedField{
			fieldName:       "StartTime",
			originFieldName: "TimeNanos",
			returnType: &primitiveType{
				structName:  "Timestamp",
				packageName: "pcommon",
				rawType:     "int64",
				defaultVal:  "0",
				testVal:     "1234567890",
			},
		},
		&messageValueField{
			fieldName:     "PeriodType",
			returnMessage: valueType,
		},
		&primitiveField{
			fieldName:  "Period",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&sliceField{
			fieldName:   "CommentStrindices",
			returnSlice: int32Slice,
		},
		&primitiveField{
			fieldName:  "DefaultSampleTypeStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveTypedField{
			fieldName:       "ProfileID",
			originFieldName: "ProfileId",
			returnType: &primitiveType{
				structName: "ProfileID",
				rawType:    "data.ProfileID",
				defaultVal: "data.ProfileID([16]byte{})",
				testVal:    "data.ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})",
			},
		},
		&sliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
		droppedAttributesCount,
		&primitiveField{
			fieldName:  "OriginalPayloadFormat",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"original payload"`,
		},
		&sliceField{
			fieldName:   "OriginalPayload",
			returnSlice: byteSlice,
		},
	},
}

var attributeUnitSlice = &sliceOfPtrs{
	structName: "AttributeUnitSlice",
	element:    attributeUnit,
}

var attributeUnit = &messageValueStruct{
	structName:     "AttributeUnit",
	description:    "// AttributeUnit Represents a mapping between Attribute Keys and Units.",
	originFullName: "otlpprofiles.AttributeUnit",
	fields: []baseField{
		&primitiveField{
			fieldName:  "AttributeKeyStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
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

var link = &messageValueStruct{
	structName:     "Link",
	description:    "// Link represents a pointer from a profile Sample to a trace Span.",
	originFullName: "otlpprofiles.Link",
	fields: []baseField{
		traceIDField,
		spanIDField,
	},
}

var valueTypeSlice = &sliceOfPtrs{
	structName: "ValueTypeSlice",
	element:    valueType,
}

var valueType = &messageValueStruct{
	structName:     "ValueType",
	description:    "// ValueType describes the type and units of a value, with an optional aggregation temporality.",
	originFullName: "otlpprofiles.ValueType",
	fields: []baseField{
		&primitiveField{
			fieldName:  "TypeStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
			fieldName:  "UnitStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveTypedField{
			fieldName: "AggregationTemporality",
			returnType: &primitiveType{
				structName: "AggregationTemporality",
				rawType:    "otlpprofiles.AggregationTemporality",
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

var sample = &messageValueStruct{
	structName:     "Sample",
	description:    "// Sample represents each record value encountered within a profiled program.",
	originFullName: "otlpprofiles.Sample",
	fields: []baseField{
		&primitiveField{
			fieldName:  "LocationsStartIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
			fieldName:  "LocationsLength",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&sliceField{
			fieldName:   "Value",
			returnSlice: int64Slice,
		},
		&sliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
		&optionalPrimitiveValue{
			fieldName:  "LinkIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&sliceField{
			fieldName:   "TimestampsUnixNano",
			returnSlice: uInt64Slice,
		},
	},
}

var mappingSlice = &sliceOfPtrs{
	structName: "MappingSlice",
	element:    mapping,
}

var mapping = &messageValueStruct{
	structName:     "Mapping",
	description:    "// Mapping describes the mapping of a binary in memory, including its address range, file offset, and metadata like build ID",
	originFullName: "otlpprofiles.Mapping",
	fields: []baseField{
		&primitiveField{
			fieldName:  "MemoryStart",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&primitiveField{
			fieldName:  "MemoryLimit",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&primitiveField{
			fieldName:  "FileOffset",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&primitiveField{
			fieldName:  "FilenameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&sliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
		&primitiveField{
			fieldName:  "HasFunctions",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&primitiveField{
			fieldName:  "HasFilenames",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&primitiveField{
			fieldName:  "HasLineNumbers",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&primitiveField{
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

var location = &messageValueStruct{
	structName:     "Location",
	description:    "// Location describes function and line table debug information.",
	originFullName: "otlpprofiles.Location",
	fields: []baseField{
		&optionalPrimitiveValue{
			fieldName:  "MappingIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
			fieldName:  "Address",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&sliceField{
			fieldName:   "Line",
			returnSlice: lineSlice,
		},
		&primitiveField{
			fieldName:  "IsFolded",
			returnType: "bool",
			defaultVal: "false",
			testVal:    "true",
		},
		&sliceField{
			fieldName:   "AttributeIndices",
			returnSlice: int32Slice,
		},
	},
}

var lineSlice = &sliceOfPtrs{
	structName: "LineSlice",
	element:    line,
}

var line = &messageValueStruct{
	structName:     "Line",
	description:    "// Line details a specific line in a source code, linked to a function.",
	originFullName: "otlpprofiles.Line",
	fields: []baseField{
		&primitiveField{
			fieldName:  "FunctionIndex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
			fieldName:  "Line",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
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

var function = &messageValueStruct{
	structName:     "Function",
	description:    "// Function describes a function, including its human-readable name, system name, source file, and starting line number in the source.",
	originFullName: "otlpprofiles.Function",
	fields: []baseField{
		&primitiveField{
			fieldName:  "NameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
			fieldName:  "SystemNameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
			fieldName:  "FilenameStrindex",
			returnType: "int32",
			defaultVal: "int32(0)",
			testVal:    "int32(1)",
		},
		&primitiveField{
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

var attribute = &messageValueStruct{
	structName:     "Attribute",
	description:    "// Attribute describes an attribute stored in a profile's attribute table.",
	originFullName: "v1.KeyValue",
	fields: []baseField{
		&primitiveField{
			fieldName:  "Key",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"key"`,
		},
		&messageValueField{
			fieldName:     "Value",
			returnMessage: anyValue,
		},
	},
}
