// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandExtendedType_StandardTypes_NoOp(t *testing.T) {
	standardTypes := []string{"", "string", "integer", "number", "boolean", "object", "array", "null"}
	for _, typ := range standardTypes {
		t.Run(typ, func(t *testing.T) {
			md := &ConfigMetadata{Type: typ}
			require.NoError(t, expandExtendedType(md))
			assert.Equal(t, typ, md.Type, "standard type should not be modified")
			assert.Empty(t, md.GoType)
			assert.Empty(t, md.Format)
			assert.Nil(t, md.Items)
		})
	}
}

func TestExpandExtendedType_UnknownAlias_Error(t *testing.T) {
	md := &ConfigMetadata{Type: "foobar"}
	err := expandExtendedType(md)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "foobar")
	assert.Contains(t, err.Error(), "unknown config type")
}

func TestExpandExtendedType_IntegerAliases(t *testing.T) {
	cases := []struct {
		alias  string
		goType string
	}{
		{"rune", "rune"},
		{"byte", "byte"},
		{"uint", "uint"},
		{"int8", "int8"},
		{"uint8", "uint8"},
		{"int16", "int16"},
		{"uint16", "uint16"},
		{"int32", "int32"},
		{"uint32", "uint32"},
		{"int64", "int64"},
		{"uint64", "uint64"},
	}
	for _, tc := range cases {
		t.Run(tc.alias, func(t *testing.T) {
			md := &ConfigMetadata{Type: tc.alias}
			require.NoError(t, expandExtendedType(md))
			assert.Equal(t, "integer", md.Type)
			assert.Equal(t, tc.goType, md.GoType)
			assert.Empty(t, md.Format)
			assert.Nil(t, md.Items)
		})
	}
}

func TestExpandExtendedType_NumberAliases(t *testing.T) {
	cases := []struct {
		alias  string
		goType string
	}{
		{"float32", "float32"},
		{"float64", "float64"},
	}
	for _, tc := range cases {
		t.Run(tc.alias, func(t *testing.T) {
			md := &ConfigMetadata{Type: tc.alias}
			require.NoError(t, expandExtendedType(md))
			assert.Equal(t, "number", md.Type)
			assert.Equal(t, tc.goType, md.GoType)
		})
	}
}

func TestExpandExtendedType_Duration(t *testing.T) {
	md := &ConfigMetadata{Type: "duration"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Empty(t, md.GoType, "GoType is set by enhanceTimeTypes, not here")
	assert.Equal(t, "duration", md.Format)
}

func TestExpandExtendedType_Time(t *testing.T) {
	md := &ConfigMetadata{Type: "time"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Empty(t, md.GoType)
	assert.Equal(t, "date-time", md.Format)
}

func TestExpandExtendedType_OpaqueString(t *testing.T) {
	md := &ConfigMetadata{Type: "opaque_string"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/config/configopaque.String", md.GoType)
	assert.Empty(t, md.Format)
}

func TestExpandExtendedType_ID(t *testing.T) {
	md := &ConfigMetadata{Type: "id"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/component.ID", md.GoType)
}

func TestExpandExtendedType_OpaqueMap(t *testing.T) {
	md := &ConfigMetadata{Type: "opaque_map"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "array", md.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/config/configopaque.MapList", md.GoType)
	require.NotNil(t, md.Items)
	assert.Equal(t, "object", md.Items.Type)
	assert.Contains(t, md.Items.Properties, "name")
	assert.Contains(t, md.Items.Properties, "value")
}

// TestExpandExtendedType_DoesNotClobberExplicitGoType verifies that an explicit x-customType
// on the alias node is preserved (e.g. type: int64 + x-customType: myapp.SpecialInt).
func TestExpandExtendedType_DoesNotClobberExplicitGoType(t *testing.T) {
	md := &ConfigMetadata{Type: "int64", GoType: "myapp/pkg.SpecialInt"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "integer", md.Type)
	assert.Equal(t, "myapp/pkg.SpecialInt", md.GoType, "explicit GoType must not be overwritten")
}

// TestExpandExtendedType_DoesNotClobberExplicitFormat verifies that an explicit format
// on a duration-like alias is not overwritten.
func TestExpandExtendedType_DoesNotClobberExplicitFormat(t *testing.T) {
	// opaque_string with an explicit format — should keep the custom format
	md := &ConfigMetadata{Type: "opaque_string", Format: "custom-format"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Equal(t, "custom-format", md.Format, "explicit Format must not be overwritten")
}

// TestExpandExtendedType_DoesNotClobberExplicitItems verifies that explicit Items is kept.
func TestExpandExtendedType_DoesNotClobberExplicitItems(t *testing.T) {
	existingItems := &ConfigMetadata{Type: "integer"}
	md := &ConfigMetadata{Type: "opaque_map", Items: existingItems}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "array", md.Type)
	assert.Same(t, existingItems, md.Items, "explicit Items must not be overwritten")
}
