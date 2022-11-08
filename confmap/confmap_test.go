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
	"os"
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
	assert.NoError(t, conf.Unmarshal(cfg))
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
	assert.NoError(t, conf.Unmarshal(cfg))
	assert.NotNil(t, cfg.Boolean)
	assert.True(t, *cfg.Boolean)
	assert.NotNil(t, cfg.Struct)
	assert.Equal(t, s1, cfg.Struct)
	assert.NotNil(t, cfg.MapStruct)
	// TODO: Investigate this unexpected result.
	assert.Equal(t, &Struct{}, cfg.MapStruct["struct"])
}

func TestUnmarshalWithErrorUnused(t *testing.T) {
	stringMap := map[string]interface{}{
		"boolean": true,
		"string":  "this is a string",
	}
	conf := NewFromStringMap(stringMap)
	assert.Error(t, conf.Unmarshal(&TestIDConfig{}, WithErrorUnused()))
}

type TestConfig struct {
	Boolean   *bool              `mapstructure:"boolean"`
	Struct    *Struct            `mapstructure:"struct"`
	MapStruct map[string]*Struct `mapstructure:"map_struct"`
}

func (t TestConfig) Marshal(conf *Conf) error {
	if t.Boolean != nil && !*t.Boolean {
		return errors.New("unable to marshal")
	}
	if err := conf.Marshal(t); err != nil {
		return err
	}
	return conf.Merge(NewFromStringMap(map[string]interface{}{
		"additional": "field",
	}))
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

func (tID TestID) MarshalText() (text []byte, err error) {
	out := string(tID)
	if !strings.HasSuffix(out, "_") {
		out += "_"
	}
	return []byte(out), nil
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
	assert.NoError(t, conf.Unmarshal(cfg))
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
	assert.Error(t, conf.Unmarshal(cfg))
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
	assert.Error(t, conf.Unmarshal(cfg))
}

func TestMarshal(t *testing.T) {
	conf := New()
	cfg := &TestIDConfig{
		Boolean: true,
		Map: map[TestID]string{
			"string": "this is a string",
		},
	}
	assert.NoError(t, conf.Marshal(cfg))
	assert.Equal(t, true, conf.Get("bool"))
	assert.Equal(t, map[string]interface{}{"string_": "this is a string"}, conf.Get("map"))
}

func TestMarshalDuplicateID(t *testing.T) {
	conf := New()
	cfg := &TestIDConfig{
		Boolean: true,
		Map: map[TestID]string{
			"string":  "this is a string",
			"string_": "this is another string",
		},
	}
	assert.Error(t, conf.Marshal(cfg))
}

func TestMarshalError(t *testing.T) {
	conf := New()
	assert.Error(t, conf.Marshal(nil))
}

func TestMarshaler(t *testing.T) {
	conf := New()
	cfg := &TestConfig{
		Struct: &Struct{
			Name: "StructName",
		},
	}
	assert.NoError(t, conf.Marshal(cfg))
	assert.Equal(t, "field", conf.Get("additional"))

	conf = New()
	type NestedMarshaler struct {
		TestConfig *TestConfig
	}
	nmCfg := &NestedMarshaler{
		TestConfig: cfg,
	}
	assert.NoError(t, conf.Marshal(nmCfg))
	sub, err := conf.Sub("testconfig")
	assert.NoError(t, err)
	assert.True(t, sub.IsSet("additional"))
	assert.Equal(t, "field", sub.Get("additional"))
	varBool := false
	nmCfg.TestConfig.Boolean = &varBool
	assert.Error(t, conf.Marshal(nmCfg))
}

// newConfFromFile creates a new Conf by reading the given file.
func newConfFromFile(t testing.TB, fileName string) map[string]interface{} {
	content, err := os.ReadFile(filepath.Clean(fileName))
	require.NoErrorf(t, err, "unable to read the file %v", fileName)

	var data map[string]interface{}
	require.NoError(t, yaml.Unmarshal(content, &data), "unable to parse yaml")

	return NewFromStringMap(data).ToStringMap()
}

type testConfig struct {
	Next    *nextConfig `mapstructure:"next"`
	Another string      `mapstructure:"another"`
}

func (tc *testConfig) Unmarshal(component *Conf) error {
	if err := component.Unmarshal(tc); err != nil {
		return err
	}
	tc.Another += " is only called directly"
	return nil
}

type nextConfig struct {
	String  string `mapstructure:"string"`
	private string
}

func (nc *nextConfig) Unmarshal(component *Conf) error {
	if err := component.Unmarshal(nc); err != nil {
		return err
	}
	nc.String += " is called"
	return nil
}

func TestUnmarshaler(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]interface{}{
		"next": map[string]interface{}{
			"string": "make sure this",
		},
		"another": "make sure this",
	})

	tc := &testConfig{}
	assert.NoError(t, cfgMap.Unmarshal(tc))
	assert.Equal(t, "make sure this", tc.Another)
	assert.Equal(t, "make sure this is called", tc.Next.String)
}

func TestUnmarshalerKeepAlreadyInitialized(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]interface{}{
		"next": map[string]interface{}{
			"string": "make sure this",
		},
		"another": "make sure this",
	})

	tc := &testConfig{Next: &nextConfig{
		private: "keep already configured members",
	}}
	assert.NoError(t, cfgMap.Unmarshal(tc))
	assert.Equal(t, "make sure this", tc.Another)
	assert.Equal(t, "make sure this is called", tc.Next.String)
	assert.Equal(t, "keep already configured members", tc.Next.private)
}

func TestDirectUnmarshaler(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]interface{}{
		"next": map[string]interface{}{
			"string": "make sure this",
		},
		"another": "make sure this",
	})

	tc := &testConfig{Next: &nextConfig{
		private: "keep already configured members",
	}}
	assert.NoError(t, tc.Unmarshal(cfgMap))
	assert.Equal(t, "make sure this is only called directly", tc.Another)
	assert.Equal(t, "make sure this is called", tc.Next.String)
	assert.Equal(t, "keep already configured members", tc.Next.private)
}

type testErrConfig struct {
	Err errConfig `mapstructure:"err"`
}

func (tc *testErrConfig) Unmarshal(component *Conf) error {
	return component.Unmarshal(tc)
}

type errConfig struct {
	Foo string `mapstructure:"foo"`
}

func (tc *errConfig) Unmarshal(component *Conf) error {
	return errors.New("never works")
}

func TestUnmarshalerErr(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]interface{}{
		"err": map[string]interface{}{
			"foo": "will not unmarshal due to error",
		},
	})

	expectErr := "1 error(s) decoding:\n\n* error decoding 'err': never works"

	tc := &testErrConfig{}
	assert.EqualError(t, cfgMap.Unmarshal(tc), expectErr)
	assert.Empty(t, tc.Err.Foo)
}
