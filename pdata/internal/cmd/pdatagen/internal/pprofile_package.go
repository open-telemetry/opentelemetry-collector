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
	},
}

var byteSliceType = &primitiveType{
	structName: "[]byte",
	rawType:    "[]byte",
	defaultVal: "[]byte(nil)",
	testVal:    `[]byte("test")`,
}
