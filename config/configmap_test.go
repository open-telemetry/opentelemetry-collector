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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToStringMap_WithSet(t *testing.T) {
	parser := NewMap()
	parser.Set("key::embedded", int64(123))
	assert.Equal(t, map[string]interface{}{"key": map[string]interface{}{"embedded": int64(123)}}, parser.ToStringMap())
}

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		stringMap map[string]interface{}
	}{
		{
			name:     "Sample Collector configuration",
			fileName: "testdata/config.yaml",
			stringMap: map[string]interface{}{
				"receivers": map[string]interface{}{
					"nop":            nil,
					"nop/myreceiver": nil,
				},

				"processors": map[string]interface{}{
					"nop":             nil,
					"nop/myprocessor": nil,
				},

				"exporters": map[string]interface{}{
					"nop":            nil,
					"nop/myexporter": nil,
				},

				"extensions": map[string]interface{}{
					"nop":             nil,
					"nop/myextension": nil,
				},

				"service": map[string]interface{}{
					"extensions": []interface{}{"nop"},
					"pipelines": map[string]interface{}{
						"traces": map[string]interface{}{
							"receivers":  []interface{}{"nop"},
							"processors": []interface{}{"nop"},
							"exporters":  []interface{}{"nop"},
						},
					},
				},
			},
		},
		{
			name:     "Sample types",
			fileName: "testdata/basic_types.yaml",
			stringMap: map[string]interface{}{
				"typed.options": map[string]interface{}{
					"floating.point.example": 3.14,
					"integer.example":        1234,
					"bool.example":           false,
					"string.example":         "this is a string",
					"nil.example":            nil,
				},
			},
		},
		{
			name:     "Embedded keys",
			fileName: "testdata/embedded_keys.yaml",
			stringMap: map[string]interface{}{
				"typed": map[string]interface{}{"options": map[string]interface{}{
					"floating": map[string]interface{}{"point": map[string]interface{}{"example": 3.14}},
					"integer":  map[string]interface{}{"example": 1234},
					"bool":     map[string]interface{}{"example": false},
					"string":   map[string]interface{}{"example": "this is a string"},
					"nil":      map[string]interface{}{"example": nil},
				}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser, err := NewMapFromFile(test.fileName)
			require.NoError(t, err, "Unable to read configuration file '%s'", test.fileName)
			assert.Equal(t, test.stringMap, parser.ToStringMap())
		})
	}
}

func TestExpandNilStructPointersFunc(t *testing.T) {
	stringMap := map[string]interface{}{
		"boolean": nil,
		"struct":  nil,
		"map_struct": map[string]interface{}{
			"struct": nil,
		},
	}
	parser := NewMapFromStringMap(stringMap)
	cfg := &TestConfig{}
	assert.Nil(t, cfg.Struct)
	assert.NoError(t, parser.UnmarshalExact(cfg))
	assert.NotNil(t, cfg.Boolean)
	assert.False(t, *cfg.Boolean)
	assert.NotNil(t, cfg.Struct)
	assert.NotNil(t, cfg.MapStruct)
	assert.Equal(t, &Struct{}, cfg.MapStruct["struct"])
}

func TestExpandNilStructPointersFunc_ToStringMap(t *testing.T) {
	stringMap := map[string]interface{}{
		"boolean":   nil,
		"struct":    nil,
		"interface": nil,
		"map_struct": map[string]interface{}{
			"struct": nil,
		},
	}
	parser := NewMapFromStringMap(stringMap)
	cfg := make(map[string]interface{})
	assert.NoError(t, parser.UnmarshalExact(&cfg))
	assert.Contains(t, cfg, "boolean")
	assert.Nil(t, cfg["boolean"])
	assert.Contains(t, cfg, "struct")
	assert.Nil(t, cfg["struct"])
	assert.Contains(t, cfg, "map_struct")
	assert.Contains(t, cfg["map_struct"], "struct")
	assert.Nil(t, cfg["map_struct"].(map[string]interface{})["struct"])
}

func TestExpandNilStructPointersFunc_DefaultNotNilConfigNil(t *testing.T) {
	stringMap := map[string]interface{}{
		"boolean":   nil,
		"struct":    nil,
		"interface": nil,
		"map_struct": map[string]interface{}{
			"struct": nil,
		},
	}
	parser := NewMapFromStringMap(stringMap)
	varBool := true
	cfg := &TestConfig{
		Boolean:   &varBool,
		Struct:    &Struct{},
		MapStruct: map[string]*Struct{"struct": {}},
	}
	assert.NoError(t, parser.UnmarshalExact(cfg))
	assert.NotNil(t, cfg.Boolean)
	assert.True(t, *cfg.Boolean)
	assert.NotNil(t, cfg.Interface)
	assert.NotNil(t, cfg.Struct)
	assert.NotNil(t, cfg.MapStruct)
	assert.Equal(t, &Struct{}, cfg.MapStruct["struct"])
}

type TestConfig struct {
	Boolean   *bool              `mapstructure:"boolean"`
	Interface interface{}        `mapstructure:"interface"`
	Struct    *Struct            `mapstructure:"struct"`
	MapStruct map[string]*Struct `mapstructure:"map_struct"`
}

type Struct struct{}
