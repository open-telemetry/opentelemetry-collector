// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var pprofile = &Package{
	name: "pprofile",
	path: "pprofile",
	imports: []string{
		`"go.opentelemetry.io/collector/pdata/internal"`,
		`"go.opentelemetry.io/collector/pdata/internal/data"`,
		`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1experimental"`,
		`"go.opentelemetry.io/collector/pdata/pcommon"`,
	},
	testImports: []string{
		`"testing"`,
		`"unsafe"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`"go.opentelemetry.io/collector/pdata/internal"`,
		`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1experimental"`,
		`"go.opentelemetry.io/collector/pdata/pcommon"`,
	},
	structs: []baseStruct{
		resourceProfilesSlice,
		resourceProfiles,
		scopeProfilesSlice,
		scopeProfiles,
		profilesContainersSlice,
		profileContainer,
		profile,
		valueTypeSlice,
		valueType,
		sampleSlice,
		sample,
		labelSlice,
		label,
		mappingSlice,
		mapping,
		locationSlice,
		location,
		lineSlice,
		line,
		functionSlice,
		function,
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
			returnSlice: profilesContainersSlice,
		},
	},
}

var profilesContainersSlice = &sliceOfPtrs{
	structName: "ProfilesContainersSlice",
	element:    profileContainer,
}

var profileContainer = &messageValueStruct{
	structName:     "ProfileContainer",
	description:    "// ProfileContainer are an experimental implementation of the OpenTelemetry Profiles Data Model.\n",
	originFullName: "otlpprofiles.ProfileContainer",
	fields: []baseField{
		&sliceField{
			fieldName:       "ProfileID",
			originFieldName: "ProfileId",
			returnSlice:     byteSlice,
		},
		&primitiveTypedField{
			fieldName:       "StartTime",
			originFieldName: "StartTimeUnixNano",
			returnType:      timestampType,
		},
		&primitiveTypedField{
			fieldName:       "EndTime",
			originFieldName: "EndTimeUnixNano",
			returnType:      timestampType,
		},
		attributes,
		droppedAttributesCount,
		&messageValueField{
			fieldName:     "Profile",
			returnMessage: profile,
		},
	},
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
			fieldName:   "Mapping",
			returnSlice: mappingSlice,
		},
		&sliceField{
			fieldName:   "Location",
			returnSlice: locationSlice,
		},
		&sliceField{
			fieldName:   "LocationIndices",
			returnSlice: int64Slice,
		},
		&sliceField{
			fieldName:   "Function",
			returnSlice: functionSlice,
		},
		&sliceField{
			fieldName:   "AttributeTable",
			returnSlice: mapStruct,
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
		&primitiveField{
			fieldName:  "DropFrames",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "KeepFrames",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
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
			fieldName:   "Comment",
			returnSlice: int64Slice,
		},
		&primitiveField{
			fieldName:  "DefaultSampleType",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
	},
}

var valueTypeSlice = &sliceOfValues{
	structName: "ValueTypeSlice",
	element:    valueType,
}

var valueType = &messageValueStruct{
	structName:     "ValueType",
	description:    "// ValueType describes the type and units of a value, with an optional aggregation temporality.",
	originFullName: "otlpprofiles.ValueType",
	fields: []baseField{
		&primitiveField{
			fieldName:  "Type",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "Unit",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "AggregationTemporality",
			returnType: "otlpprofiles.AggregationTemporality",
			defaultVal: "otlpprofiles.AggregationTemporality(0)",
			testVal:    "otlpprofiles.AggregationTemporality(1)",
		},
	},
}

var sampleSlice = &sliceOfValues{
	structName: "SampleSlice",
	element:    sample,
}

var sample = &messageValueStruct{
	structName:     "Sample",
	description:    "// Sample represents each record value encountered within a profiled program.",
	originFullName: "otlpprofiles.Sample",
	fields: []baseField{
		&sliceField{
			fieldName:   "LocationIndex",
			returnSlice: uInt64Slice,
		},
		&primitiveField{
			fieldName:  "LocationsStartIndex",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&primitiveField{
			fieldName:  "LocationsLength",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&primitiveField{
			fieldName:  "StacktraceIdIndex",
			returnType: "uint32",
			defaultVal: "uint32(0)",
			testVal:    "uint32(1)",
		},
		&sliceField{
			fieldName:   "Value",
			returnSlice: int64Slice,
		},
		&sliceField{
			fieldName:   "Label",
			returnSlice: labelSlice,
		},
		&sliceField{
			fieldName:   "Attributes",
			returnSlice: uInt64Slice,
		},
		&primitiveField{
			fieldName:  "Link",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
		},
		&sliceField{
			fieldName:   "TimestampsUnixNano",
			returnSlice: uInt64Slice,
		},
	},
}

var labelSlice = &sliceOfValues{
	structName: "LabelSlice",
	element:    label,
}

var label = &messageValueStruct{
	structName:     "Label",
	description:    "// Label provided additional context for a sample",
	originFullName: "otlpprofiles.Label",
	fields: []baseField{
		&primitiveField{
			fieldName:  "Key",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "Str",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "Num",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "NumUnit",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
	},
}

var mappingSlice = &sliceOfValues{
	structName: "MappingSlice",
	element:    mapping,
}

var mapping = &messageValueStruct{
	structName:     "Mapping",
	description:    "// Mapping describes the mapping of a binary in memory, including its address range, file offset, and metadata like build ID",
	originFullName: "otlpprofiles.Mapping",
	fields: []baseField{
		&primitiveField{
			fieldName:       "ID",
			originFieldName: "Id",
			returnType:      "uint64",
			defaultVal:      "uint64(0)",
			testVal:         "uint64(1)",
		},
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
			fieldName:  "Filename",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:       "BuildID",
			originFieldName: "BuildId",
			returnType:      "int64",
			defaultVal:      "int64(0)",
			testVal:         "int64(1)",
		},
		&primitiveField{
			fieldName:       "BuildIDKind",
			originFieldName: "BuildIdKind",
			returnType:      "otlpprofiles.BuildIdKind",
			defaultVal:      "otlpprofiles.BuildIdKind(0)",
			testVal:         "otlpprofiles.BuildIdKind(1)",
		},
		&sliceField{
			fieldName:   "Attributes",
			returnSlice: uInt64Slice,
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

var locationSlice = &sliceOfValues{
	structName: "LocationSlice",
	element:    location,
}
var location = &messageValueStruct{
	structName:     "Location",
	description:    "// Location describes function and line table debug information.",
	originFullName: "otlpprofiles.Location",
	fields: []baseField{
		&primitiveField{
			fieldName:       "ID",
			originFieldName: "Id",
			returnType:      "uint64",
			defaultVal:      "uint64(0)",
			testVal:         "uint64(1)",
		},
		&primitiveField{
			fieldName:  "MappingIndex",
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
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
		&primitiveField{
			fieldName:  "TypeIndex",
			returnType: "uint32",
			defaultVal: "uint32(0)",
			testVal:    "uint32(1)",
		},
		&sliceField{
			fieldName:   "Attributes",
			returnSlice: uInt64Slice,
		},
	},
}

var lineSlice = &sliceOfValues{
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
			returnType: "uint64",
			defaultVal: "uint64(0)",
			testVal:    "uint64(1)",
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

var functionSlice = &sliceOfValues{
	structName: "FunctionSlice",
	element:    function,
}

var function = &messageValueStruct{
	structName:     "Function",
	description:    "// Function describes a function, including its human-readable name, system name, source file, and starting line number in the source.",
	originFullName: "otlpprofiles.Function",
	fields: []baseField{
		&primitiveField{
			fieldName:       "ID",
			originFieldName: "Id",
			returnType:      "uint64",
			defaultVal:      "uint64(0)",
			testVal:         "uint64(1)",
		},
		&primitiveField{
			fieldName:  "Name",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "SystemName",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "Filename",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "StartLine",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
	},
}

var attributeUnitSlice = &sliceOfValues{
	structName: "AttributeUnitSlice",
	element:    attributeUnit,
}

var attributeUnit = &messageValueStruct{
	structName:     "AttributeUnit",
	description:    "// AttributeUnit Represents a mapping between Attribute Keys and Units.",
	originFullName: "otlpprofiles.AttributeUnit",
	fields: []baseField{
		&primitiveField{
			fieldName:  "AttributeKey",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
		&primitiveField{
			fieldName:  "Unit",
			returnType: "int64",
			defaultVal: "int64(0)",
			testVal:    "int64(1)",
		},
	},
}

var linkSlice = &sliceOfValues{
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
