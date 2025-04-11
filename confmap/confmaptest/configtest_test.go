// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmaptest

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestLoadConfFileNotFound(t *testing.T) {
	_, err := LoadConf("file/not/found")
	assert.Error(t, err)
}

func TestLoadConfInvalidYAML(t *testing.T) {
	_, err := LoadConf(filepath.Join("testdata", "invalid.yaml"))
	require.Error(t, err)
}

func TestLoadConf(t *testing.T) {
	cfg, err := LoadConf(filepath.Join("testdata", "simple.yaml"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"floating": 3.14}, cfg.ToStringMap())
}

func TestToStringMapSanitizeEmptySlice(t *testing.T) {
	cfg, err := LoadConf(filepath.Join("testdata", "empty-slice.yaml"))
	require.NoError(t, err)
	var nilSlice []any
	assert.Equal(t, map[string]any{"slice": nilSlice}, cfg.ToStringMap())
}

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, ValidateProviderScheme(&schemeProvider{scheme: "file"}))
	assert.NoError(t, ValidateProviderScheme(&schemeProvider{scheme: "s3"}))
	assert.NoError(t, ValidateProviderScheme(&schemeProvider{scheme: "a.l-l+"}))
	// Too short.
	require.Error(t, ValidateProviderScheme(&schemeProvider{scheme: "a"}))
	// Invalid first character.
	require.Error(t, ValidateProviderScheme(&schemeProvider{scheme: "3s"}))
	// Invalid underscore character.
	assert.Error(t, ValidateProviderScheme(&schemeProvider{scheme: "all_"}))
}

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
				require.ErrorContains(t, err, tt.wantErrMsgSubStr)
			}
		})
	}
}

type schemeProvider struct {
	scheme string
}

func (s schemeProvider) Retrieve(context.Context, string, confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return nil, nil
}

func (s schemeProvider) Scheme() string {
	return s.scheme
}

func (s schemeProvider) Shutdown(context.Context) error {
	return nil
}
