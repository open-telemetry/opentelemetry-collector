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
		valueType,
		valueTypes,
		samples,
		sample,
		labels,
		label,
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
		&primitiveTypedField{
			fieldName:       "ProfileID",
			originFieldName: "ProfileId",
			returnType:      byteSliceType,
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
			returnSlice: valueTypes,
		},
		&sliceField{
			fieldName:   "Sample",
			returnSlice: samples,
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
	},
}

var byteSliceType = &primitiveType{
	structName: "[]byte",
	rawType:    "[]byte",
	defaultVal: "[]byte(nil)",
	testVal:    `[]byte("test")`,
}

var valueTypes = &sliceOfValues{
	structName: "ValueTypes",
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

var samples = &sliceOfValues{
	structName: "Samples",
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
			returnSlice: labels,
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

var labels = &sliceOfValues{
	structName: "Labels",
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
