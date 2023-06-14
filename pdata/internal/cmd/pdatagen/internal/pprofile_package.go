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
		profile,
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
			returnSlice: profileSlice,
		},
	},
}

var profileSlice = &sliceOfPtrs{
	structName: "ProfileSlice",
	element:    profile,
}

var profile = &messageValueStruct{
	structName:     "Profile",
	description:    "// Profile are experimental implementation of OpenTelemetry Profile Data Model.\n",
	originFullName: "otlpprofiles.Profile",
	fields: []baseField{
		&primitiveTypedField{
			fieldName:       "ProfileID",
			originFieldName: "ProfileId",
			returnType:      profileIDType,
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
	},
}
