// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var pentity = &Package{
	info: &PackageInfo{
		name: "pentity",
		path: "pentity",
		imports: []string{
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/data"`,
			`otlpentities "go.opentelemetry.io/collector/pdata/internal/data/protogen/entities/v1"`,
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
			`otlpentities "go.opentelemetry.io/collector/pdata/internal/data/protogen/entities/v1"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		resourceEntitiesSlice,
		resourceEntities,
		scopeEntitiesSlice,
		scopeEntities,
		entityEventSlice,
		entityEvent,
		entityState,
		entityDelete,
	},
}

var resourceEntitiesSlice = &sliceOfPtrs{
	structName: "ResourceEntitiesSlice",
	element:    resourceEntities,
}

var resourceEntities = &messageValueStruct{
	structName:     "ResourceEntities",
	description:    "// ResourceEntities is a collection of entities from a Resource.",
	originFullName: "otlpentities.ResourceEntities",
	fields: []baseField{
		resourceField,
		schemaURLField,
		&sliceField{
			fieldName:   "ScopeEntities",
			returnSlice: scopeEntitiesSlice,
		},
	},
}

var scopeEntitiesSlice = &sliceOfPtrs{
	structName: "ScopeEntitiesSlice",
	element:    scopeEntities,
}

var scopeEntities = &messageValueStruct{
	structName:     "ScopeEntities",
	description:    "// ScopeEntities is a collection of entities from a LibraryInstrumentation.",
	originFullName: "otlpentities.ScopeEntities",
	fields: []baseField{
		scopeField,
		schemaURLField,
		&sliceField{
			fieldName:   "EntityEvents",
			returnSlice: entityEventSlice,
		},
	},
}

var entityEventSlice = &sliceOfPtrs{
	structName: "EntityEventSlice",
	element:    entityEvent,
}

var entityEvent = &messageValueStruct{
	structName:     "EntityEvent",
	description:    "// EntityEvent are experimental implementation of OpenTelemetry Entity Data Model.\n",
	originFullName: "otlpentities.EntityEvent",
	fields: []baseField{
		&primitiveTypedField{
			fieldName:       "Timestamp",
			originFieldName: "TimeUnixNano",
			returnType:      timestampType,
		},
		&primitiveField{
			fieldName:  "EntityType",
			returnType: "string",
			defaultVal: `""`,
			testVal:    `"service"`,
		},
		entityID,
		&oneOfField{
			typeName:                   "EventType",
			originFieldName:            "Data",
			testValueIdx:               1, // Delete
			omitOriginFieldNameInNames: true,
			values: []oneOfValue{
				&oneOfMessageValue{
					fieldName:              "EntityState",
					originFieldPackageName: "otlpentities",
					returnMessage:          entityState,
				},
				&oneOfMessageValue{
					fieldName:              "EntityDelete",
					originFieldPackageName: "otlpentities",
					returnMessage:          entityDelete,
				},
			},
		},
	},
}

var entityID = &sliceField{
	fieldName:   "Id",
	returnSlice: mapStruct,
}

var entityState = &messageValueStruct{
	structName:     "EntityState",
	description:    "// EntityState are experimental implementation of OpenTelemetry Entity Data Model.\n",
	originFullName: "otlpentities.EntityState",
	fields: []baseField{
		attributes,
		droppedAttributesCount,
	},
}

var entityDelete = &messageValueStruct{
	structName:     "EntityDelete",
	description:    "// EntityDelete are experimental implementation of OpenTelemetry Entity Data Model.\n",
	originFullName: "otlpentities.EntityDelete",
	fields:         []baseField{},
}
