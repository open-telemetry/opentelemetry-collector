// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package componenttest

import (
	"io"
	"strings"
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
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrMsgSubStr))
			}
		})
	}
}
