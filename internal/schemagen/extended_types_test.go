// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandExtendedType_StandardTypes_NoOp(t *testing.T) {
	// These are the native types the generator understands directly — expandType is a no-op for them.
	standardTypes := []string{"", "string", "bool", "int", "uint", "int8", "uint8", "int16", "uint16",
		"int32", "uint32", "int64", "uint64", "object", "slice", "map", "any"}
	for _, typ := range standardTypes {
		t.Run(typ, func(t *testing.T) {
			md := &ConfigMetadata{Type: typ}
			require.NoError(t, expandExtendedType(md))
			assert.Equal(t, typ, md.Type, "standard type should not be modified")
			assert.Empty(t, md.GoType)
			assert.Empty(t, md.Format)
			assert.Nil(t, md.Values)
		})
	}
}

func TestExpandExtendedType_UnknownAlias_NoOp(t *testing.T) {
	// expandType is a no-op for types not in the alias registry — unknown types are passed through
	// so the generator can handle them (e.g. fall through to the default case in resolveGoType).
	md := &ConfigMetadata{Type: "foobar"}
	err := expandExtendedType(md)
	require.NoError(t, err)
	assert.Equal(t, "foobar", md.Type, "unknown type should be unchanged")
}

func TestExpandExtendedType_IntegerAliases(t *testing.T) {
	// rune, byte, and the full int/uint family are now first-class types — expandType is a no-op for them.
	cases := []string{"rune", "byte", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64"}
	for _, alias := range cases {
		t.Run(alias, func(t *testing.T) {
			md := &ConfigMetadata{Type: alias}
			require.NoError(t, expandExtendedType(md))
			assert.Equal(t, alias, md.Type, "integer aliases are passed through unchanged")
			assert.Empty(t, md.GoType)
			assert.Empty(t, md.Format)
			assert.Nil(t, md.Values)
		})
	}
}

func TestExpandExtendedType_FloatAliases(t *testing.T) {
	// "float" and "double" are shorthands that expand to "float32"/"float64".
	cases := []struct {
		alias    string
		wantType string
	}{
		{"float", "float32"},
		{"double", "float64"},
	}
	for _, tc := range cases {
		t.Run(tc.alias, func(t *testing.T) {
			md := &ConfigMetadata{Type: tc.alias}
			require.NoError(t, expandExtendedType(md))
			assert.Equal(t, tc.wantType, md.Type)
			assert.Empty(t, md.GoType)
		})
	}
}

func TestExpandExtendedType_NativeFloatTypes_NoOp(t *testing.T) {
	// "float32" and "float64" are native — expandType leaves them unchanged.
	for _, typ := range []string{"float32", "float64"} {
		t.Run(typ, func(t *testing.T) {
			md := &ConfigMetadata{Type: typ}
			require.NoError(t, expandExtendedType(md))
			assert.Equal(t, typ, md.Type)
			assert.Empty(t, md.GoType)
		})
	}
}

func TestExpandExtendedType_Duration(t *testing.T) {
	md := &ConfigMetadata{Type: "duration"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Equal(t, "time.Duration", md.GoType)
}

func TestExpandExtendedType_Time(t *testing.T) {
	md := &ConfigMetadata{Type: "time"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Equal(t, "time.Time", md.GoType)
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
	md := &ConfigMetadata{Type: "component_id"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "string", md.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/component.ID", md.GoType)
}

func TestExpandExtendedType_OpaqueMap(t *testing.T) {
	md := &ConfigMetadata{Type: "opaque_map"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "map", md.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/config/configopaque.MapList", md.GoType)
	require.NotNil(t, md.Values)
	assert.Equal(t, "string", md.Values.Type)
}

// TestExpandExtendedType_DoesNotClobberExplicitGoType verifies that an explicit x-customType
// on the alias node is preserved (e.g. type: int64 + x-customType: myapp.SpecialInt).
func TestExpandExtendedType_DoesNotClobberExplicitGoType(t *testing.T) {
	// "float" expands to "float32" but must not overwrite an explicit GoType.
	md := &ConfigMetadata{Type: "float", GoType: "myapp/pkg.SpecialFloat"}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "float32", md.Type)
	assert.Equal(t, "myapp/pkg.SpecialFloat", md.GoType, "explicit GoType must not be overwritten")
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

// TestExpandExtendedType_DoesNotClobberExplicitValueType verifies that an explicit ValueType is kept.
func TestExpandExtendedType_DoesNotClobberExplicitValueType(t *testing.T) {
	existingValueType := &ConfigMetadata{Type: "int"}
	md := &ConfigMetadata{Type: "opaque_map", Values: existingValueType}
	require.NoError(t, expandExtendedType(md))
	assert.Equal(t, "map", md.Type)
	assert.Same(t, existingValueType, md.Values, "explicit ValueType must not be overwritten")
}
