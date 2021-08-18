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

package configparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToStringMap_WithSet(t *testing.T) {
	parser := NewParser()
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
			parser, err := NewParserFromFile(test.fileName)
			require.NoError(t, err, "Unable to read configuration file '%s'", test.fileName)
			assert.Equal(t, test.stringMap, parser.ToStringMap())
		})
	}
}
