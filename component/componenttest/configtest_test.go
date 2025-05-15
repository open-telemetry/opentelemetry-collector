// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckConfigStructPointerAndValue(t *testing.T) {
	config := struct {
		SomeFiled string `mapstructure:"test"`
	}{}
	assert.NoError(t, CheckConfigStruct(config))
	assert.NoError(t, CheckConfigStruct(&config))
}

func TestCheckConfigStruct(t *testing.T) {
	type BadConfigTag struct {
		BadTagField int `mapstructure:"test-dash"`
	}

	tests := []struct {
		name             string
		config           any
		wantErrMsgSubStr string
	}{
		{
			name: "typical_config",
			config: struct {
				MyPublicString string `mapstructure:"string"`
			}{},
		},
		{
			name: "private_fields_ignored",
			config: struct {
				// A public type with proper tag.
				MyPublicString string `mapstructure:"string"`
				// A public type with proper tag.
				MyPublicInt string `mapstructure:"int"`
				// A public type that should be ignored.
				MyFunc func() error
				// A public type that should be ignored.
				Reader io.Reader
				// private type not tagged.
				myPrivateString string
				_someInt        int
			}{},
		},
		{
			name: "remain_mapstructure_tag",
			config: struct {
				AdditionalProperties map[string]any `mapstructure:",remain"`
			}{},
		},
		{
			name: "remain_with_interface_type",
			config: struct {
				AdditionalProperties any `mapstructure:",remain"`
			}{},
		},
		{
			name: "omitempty_mapstructure_tag",
			config: struct {
				MyPublicString string `mapstructure:",omitempty"`
			}{},
		},
		{
			name: "named_omitempty_mapstructure_tag",
			config: struct {
				MyPublicString string `mapstructure:"my_public_string,omitempty"`
			}{},
		},
		{
			name: "not_struct_nor_pointer",
			config: func(x int) int {
				return x * x
			},
			wantErrMsgSubStr: "config must be a struct or a pointer to one, the passed object is a func",
		},
		{
			name: "squash_on_non_struct",
			config: struct {
				MyInt int `mapstructure:",squash"`
			}{},
			wantErrMsgSubStr: "attempt to squash non-struct type on field \"MyInt\"",
		},
		{
			name: "remain_on_non_map",
			config: struct {
				AdditionalProperties string `mapstructure:",remain"`
			}{},
			wantErrMsgSubStr: `attempt to use "remain" on non-map or interface type field "AdditionalProperties"`,
		},
		{
			name: "bad_custom_field_name",
			config: struct {
				AdditionalProperties any `mapstructure:"Additional_Properties"`
			}{},
			wantErrMsgSubStr: `field "AdditionalProperties" has config tag "Additional_Properties" which doesn't satisfy "^[a-z0-9][a-z0-9_]*$"`,
		},
		{
			name:             "invalid_tag_detected",
			config:           BadConfigTag{},
			wantErrMsgSubStr: "field \"BadTagField\" has config tag \"test-dash\" which doesn't satisfy",
		},
		{
			name: "public_field_must_have_tag",
			config: struct {
				PublicFieldWithoutMapstructureTag string
			}{},
			wantErrMsgSubStr: "mapstructure tag not present on field \"PublicFieldWithoutMapstructureTag\"",
		},
		{
			name: "public_field_must_have_nonempty_tag",
			config: struct {
				PublicFieldWithoutMapstructureTag string `mapstructure:""`
			}{},
			wantErrMsgSubStr: "mapstructure tag on field \"PublicFieldWithoutMapstructureTag\" is empty",
		},
		{
			name: "invalid_map_item",
			config: struct {
				Map map[string]BadConfigTag `mapstructure:"test_map"`
			}{},
			wantErrMsgSubStr: "field \"BadTagField\" has config tag \"test-dash\" which doesn't satisfy",
		},
		{
			name: "invalid_slice_item",
			config: struct {
				Slice []BadConfigTag `mapstructure:"test_slice"`
			}{},
			wantErrMsgSubStr: "field \"BadTagField\" has config tag \"test-dash\" which doesn't satisfy",
		},
		{
			name: "invalid_array_item",
			config: struct {
				Array [2]BadConfigTag `mapstructure:"test_array"`
			}{},
			wantErrMsgSubStr: "field \"BadTagField\" has config tag \"test-dash\" which doesn't satisfy",
		},
		{
			name: "invalid_map_item_ptr",
			config: struct {
				Map map[string]*BadConfigTag `mapstructure:"test_map"`
			}{},
			wantErrMsgSubStr: "field \"BadTagField\" has config tag \"test-dash\" which doesn't satisfy",
		},
		{
			name: "invalid_slice_item_ptr",
			config: struct {
				Slice []*BadConfigTag `mapstructure:"test_slice"`
			}{},
			wantErrMsgSubStr: "field \"BadTagField\" has config tag \"test-dash\" which doesn't satisfy",
		},
		{
			name: "invalid_array_item_ptr",
			config: struct {
				Array [2]*BadConfigTag `mapstructure:"test_array"`
			}{},
			wantErrMsgSubStr: "field \"BadTagField\" has config tag \"test-dash\" which doesn't satisfy",
		},
		{
			name: "valid_map_item",
			config: struct {
				Map map[string]int `mapstructure:"test_map"`
			}{},
		},
		{
			name: "valid_slice_item",
			config: struct {
				Slice []string `mapstructure:"test_slice"`
			}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckConfigStruct(tt.config)
			if tt.wantErrMsgSubStr == "" {
				assert.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErrMsgSubStr)
			}
		})
	}
}
