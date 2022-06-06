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

package confmap

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestToStringMapFlatten(t *testing.T) {
	conf := NewFromStringMap(map[string]interface{}{"key::embedded": int64(123)})
	assert.Equal(t, map[string]interface{}{"key": map[string]interface{}{"embedded": int64(123)}}, conf.ToStringMap())
}

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		stringMap map[string]interface{}
	}{
		{
			name:     "Sample Collector configuration",
			fileName: filepath.Join("testdata", "config.yaml"),
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
			fileName: filepath.Join("testdata", "basic_types.yaml"),
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
			fileName: filepath.Join("testdata", "embedded_keys.yaml"),
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
			assert.Equal(t, test.stringMap, newConfFromFile(t, test.fileName))
		})
	}
}

func TestExpandNilStructPointersHookFunc(t *testing.T) {
	stringMap := map[string]interface{}{
		"boolean": nil,
		"struct":  nil,
		"map_struct": map[string]interface{}{
			"struct": nil,
		},
	}
	conf := NewFromStringMap(stringMap)
	cfg := &TestConfig{}
	assert.Nil(t, cfg.Struct)
	assert.NoError(t, conf.UnmarshalExact(cfg))
	assert.Nil(t, cfg.Boolean)
	// assert.False(t, *cfg.Boolean)
	assert.Nil(t, cfg.Struct)
	assert.NotNil(t, cfg.MapStruct)
	// TODO: Investigate this unexpected result.
	assert.Equal(t, &Struct{}, cfg.MapStruct["struct"])
}

func TestExpandNilStructPointersHookFuncDefaultNotNilConfigNil(t *testing.T) {
	stringMap := map[string]interface{}{
		"boolean": nil,
		"struct":  nil,
		"map_struct": map[string]interface{}{
			"struct": nil,
		},
	}
	conf := NewFromStringMap(stringMap)
	varBool := true
	s1 := &Struct{Name: "s1"}
	s2 := &Struct{Name: "s2"}
	cfg := &TestConfig{
		Boolean:   &varBool,
		Struct:    s1,
		MapStruct: map[string]*Struct{"struct": s2},
	}
	assert.NoError(t, conf.UnmarshalExact(cfg))
	assert.NotNil(t, cfg.Boolean)
	assert.True(t, *cfg.Boolean)
	assert.NotNil(t, cfg.Struct)
	assert.Equal(t, s1, cfg.Struct)
	assert.NotNil(t, cfg.MapStruct)
	// TODO: Investigate this unexpected result.
	assert.Equal(t, &Struct{}, cfg.MapStruct["struct"])
}

type TestConfig struct {
	Boolean   *bool              `mapstructure:"boolean"`
	Struct    *Struct            `mapstructure:"struct"`
	MapStruct map[string]*Struct `mapstructure:"map_struct"`
}

type Struct struct {
	Name string
}

type TestID string

func (tID *TestID) UnmarshalText(text []byte) error {
	*tID = TestID(strings.TrimSuffix(string(text), "_"))
	if *tID == "error" {
		return errors.New("parsing error")
	}
	return nil
}

type TestIDConfig struct {
	Boolean bool              `mapstructure:"bool"`
	Map     map[TestID]string `mapstructure:"map"`
}

func TestMapKeyStringToMapKeyTextUnmarshalerHookFunc(t *testing.T) {
	stringMap := map[string]interface{}{
		"bool": true,
		"map": map[string]interface{}{
			"string": "this is a string",
		},
	}
	conf := NewFromStringMap(stringMap)
	cfg := &TestIDConfig{}
	assert.NoError(t, conf.UnmarshalExact(cfg))
	assert.True(t, cfg.Boolean)
	assert.Equal(t, map[TestID]string{"string": "this is a string"}, cfg.Map)
}

func TestMapKeyStringToMapKeyTextUnmarshalerHookFuncDuplicateID(t *testing.T) {
	stringMap := map[string]interface{}{
		"bool": true,
		"map": map[string]interface{}{
			"string":  "this is a string",
			"string_": "this is another string",
		},
	}
	conf := NewFromStringMap(stringMap)
	cfg := &TestIDConfig{}
	assert.Error(t, conf.UnmarshalExact(cfg))
}

func TestMapKeyStringToMapKeyTextUnmarshalerHookFuncErrorUnmarshal(t *testing.T) {
	stringMap := map[string]interface{}{
		"bool": true,
		"map": map[string]interface{}{
			"error": "this is a string",
		},
	}
	conf := NewFromStringMap(stringMap)
	cfg := &TestIDConfig{}
	assert.Error(t, conf.UnmarshalExact(cfg))
}

// newConfFromFile creates a new Conf by reading the given file.
func newConfFromFile(t testing.TB, fileName string) map[string]interface{} {
	content, err := ioutil.ReadFile(filepath.Clean(fileName))
	require.NoErrorf(t, err, "unable to read the file %v", fileName)

	var data map[string]interface{}
	require.NoError(t, yaml.Unmarshal(content, &data), "unable to parse yaml")

	return NewFromStringMap(data).ToStringMap()
}
