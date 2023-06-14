// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var pprofile = &Package{
	name: "pprofile",
	path: "pprofile",
	imports: []string{
		`"sort"`,
		``,
		`"go.opentelemetry.io/collector/pdata/internal"`,
		`"go.opentelemetry.io/collector/pdata/internal/data"`,
		`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"`,
		`"go.opentelemetry.io/collector/pdata/pcommon"`,
	},
	testImports: []string{
		`"testing"`,
		`"unsafe"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`"go.opentelemetry.io/collector/pdata/internal"`,
		`"go.opentelemetry.io/collector/pdata/internal/data"`,
		`otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"`,
		`"go.opentelemetry.io/collector/pdata/pcommon"`,
	},
	structs: []baseStruct{
		resourceProfilesSlice,
		resourceProfiles,
		scopeProfilesSlice,
		scopeProfiles,
		profileSlice,
		profileRecord,
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
			fieldName:   "ProfileRecords",
			returnSlice: profileSlice,
		},
	},
}

var profileSlice = &sliceOfPtrs{
	structName: "ProfileRecordSlice",
	element:    profileRecord,
}

var profileRecord = &messageValueStruct{
	structName:     "ProfileRecord",
	description:    "// ProfileRecord are experimental implementation of OpenTelemetry Profile Data Model.\n",
	originFullName: "otlpprofiles.ProfileRecord",
	fields: []baseField{
		&primitiveTypedField{
			fieldName:       "ObservedTimestamp",
			originFieldName: "ObservedTimeUnixNano",
			returnType:      timestampType,
		},
		&primitiveTypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			returnType:      timestampType,
		},
		traceIDField,
		spanIDField,
		&primitiveTypedField{
			fieldName: "Flags",
			returnType: &primitiveType{
				structName: "ProfileRecordFlags",
				rawType:    "uint32",
				defaultVal: "0",
				testVal:    "1",
			},
		},
		&primitiveField{
			fieldName:  "SeverityText",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"INFO"`,
		},
		&primitiveTypedField{
			fieldName: "SeverityNumber",
			returnType: &primitiveType{
				structName: "SeverityNumber",
				rawType:    "otlpprofiles.SeverityNumber",
				defaultVal: `otlpprofiles.SeverityNumber(0)`,
				testVal:    `otlpprofiles.SeverityNumber(5)`,
			},
		},
		bodyField,
		attributes,
		droppedAttributesCount,
	},
}
