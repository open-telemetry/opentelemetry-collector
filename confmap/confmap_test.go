// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestToStringMapFlatten(t *testing.T) {
	conf := NewFromStringMap(map[string]any{"key::embedded": int64(123)})
	assert.Equal(t, map[string]any{"key": map[string]any{"embedded": int64(123)}}, conf.ToStringMap())
}

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		stringMap map[string]any
	}{
		{
			name:     "Sample Collector configuration",
			fileName: filepath.Join("testdata", "config.yaml"),
			stringMap: map[string]any{
				"receivers": map[string]any{
					"nop":            nil,
					"nop/myreceiver": nil,
				},

				"processors": map[string]any{
					"nop":             nil,
					"nop/myprocessor": nil,
				},

				"exporters": map[string]any{
					"nop":            nil,
					"nop/myexporter": nil,
				},

				"extensions": map[string]any{
					"nop":             nil,
					"nop/myextension": nil,
				},

				"service": map[string]any{
					"extensions": []any{"nop"},
					"pipelines": map[string]any{
						"traces": map[string]any{
							"receivers":  []any{"nop"},
							"processors": []any{"nop"},
							"exporters":  []any{"nop"},
						},
					},
				},
			},
		},
		{
			name:     "Sample types",
			fileName: filepath.Join("testdata", "basic_types.yaml"),
			stringMap: map[string]any{
				"typed.options": map[string]any{
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
			stringMap: map[string]any{
				"typed": map[string]any{"options": map[string]any{
					"floating": map[string]any{"point": map[string]any{"example": 3.14}},
					"integer":  map[string]any{"example": 1234},
					"bool":     map[string]any{"example": false},
					"string":   map[string]any{"example": "this is a string"},
					"nil":      map[string]any{"example": nil},
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
	stringMap := map[string]any{
		"boolean": nil,
		"struct":  nil,
		"map_struct": map[string]any{
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
	stringMap := map[string]any{
		"boolean": nil,
		"struct":  nil,
		"map_struct": map[string]any{
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

func TestUnmarshalWithIgnoreUnused(t *testing.T) {
	stringMap := map[string]any{
		"boolean": true,
		"string":  "this is a string",
	}
	conf := NewFromStringMap(stringMap)
	assert.Error(t, conf.Unmarshal(&TestIDConfig{}))
	assert.NoError(t, conf.Unmarshal(&TestIDConfig{}, WithIgnoreUnused()))
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
	return conf.Merge(NewFromStringMap(map[string]any{
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
	stringMap := map[string]any{
		"bool": true,
		"map": map[string]any{
			"string": "this is a string",
		},
	}
	conf := NewFromStringMap(stringMap)

	cfg := &TestIDConfig{}
	assert.NoError(t, conf.Unmarshal(cfg))
	assert.True(t, cfg.Boolean)
	assert.Equal(t, map[TestID]string{"string": "this is a string"}, cfg.Map)
}

type UintConfig struct {
	UintTest uint32 `mapstructure:"uint_test"`
}

func TestUintUnmarshalerSuccess(t *testing.T) {
	tests := []struct {
		name      string
		testValue int
	}{
		{
			name:      "Test convert 0 to uint",
			testValue: 0,
		},
		{
			name:      "Test positive uint conversion",
			testValue: 1000,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			stringMap := map[string]any{
				"uint_test": tt.testValue,
			}
			conf := NewFromStringMap(stringMap)
			cfg := &UintConfig{}
			err := conf.Unmarshal(cfg)

			assert.NoError(t, err)
			assert.Equal(t, cfg.UintTest, uint32(tt.testValue))
		})
	}
}

func TestUint64Unmarshaler(t *testing.T) {
	negativeInt := -1000
	testValue := uint64(negativeInt)

	type Uint64Config struct {
		UintTest uint64 `mapstructure:"uint_test"`
	}
	stringMap := map[string]any{
		"uint_test": testValue,
	}

	conf := NewFromStringMap(stringMap)
	cfg := &Uint64Config{}
	err := conf.Unmarshal(cfg)

	assert.NoError(t, err)
	assert.Equal(t, cfg.UintTest, testValue)
}

func TestUintUnmarshalerFailure(t *testing.T) {
	testValue := -1000
	stringMap := map[string]any{
		"uint_test": testValue,
	}
	conf := NewFromStringMap(stringMap)
	cfg := &UintConfig{}
	err := conf.Unmarshal(cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("* error decoding 'uint_test': cannot convert negative value %v to an unsigned integer", testValue))
}

func TestMapKeyStringToMapKeyTextUnmarshalerHookFuncDuplicateID(t *testing.T) {
	stringMap := map[string]any{
		"bool": true,
		"map": map[string]any{
			"string":  "this is a string",
			"string_": "this is another string",
		},
	}
	conf := NewFromStringMap(stringMap)

	cfg := &TestIDConfig{}
	assert.Error(t, conf.Unmarshal(cfg))
}

func TestMapKeyStringToMapKeyTextUnmarshalerHookFuncErrorUnmarshal(t *testing.T) {
	stringMap := map[string]any{
		"bool": true,
		"map": map[string]any{
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
	assert.Equal(t, map[string]any{"string_": "this is a string"}, conf.Get("map"))
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
func newConfFromFile(t testing.TB, fileName string) map[string]any {
	content, err := os.ReadFile(filepath.Clean(fileName))
	require.NoErrorf(t, err, "unable to read the file %v", fileName)

	var data map[string]any
	require.NoError(t, yaml.Unmarshal(content, &data), "unable to parse yaml")

	return NewFromStringMap(data).ToStringMap()
}

type testConfig struct {
	Next            *nextConfig `mapstructure:"next"`
	Another         string      `mapstructure:"another"`
	EmbeddedConfig  `mapstructure:",squash"`
	EmbeddedConfig2 `mapstructure:",squash"`
}

type testConfigWithoutUnmarshaler struct {
	Next            *nextConfig `mapstructure:"next"`
	Another         string      `mapstructure:"another"`
	EmbeddedConfig  `mapstructure:",squash"`
	EmbeddedConfig2 `mapstructure:",squash"`
}

type testConfigWithEmbeddedError struct {
	Next                    *nextConfig `mapstructure:"next"`
	Another                 string      `mapstructure:"another"`
	EmbeddedConfigWithError `mapstructure:",squash"`
}

type testConfigWithMarshalError struct {
	Next                           *nextConfig `mapstructure:"next"`
	Another                        string      `mapstructure:"another"`
	EmbeddedConfigWithMarshalError `mapstructure:",squash"`
}

func (tc *testConfigWithEmbeddedError) Unmarshal(component *Conf) error {
	if err := component.Unmarshal(tc, WithIgnoreUnused()); err != nil {
		return err
	}
	return nil
}

type EmbeddedConfig struct {
	Some string `mapstructure:"some"`
}

func (ec *EmbeddedConfig) Unmarshal(component *Conf) error {
	if err := component.Unmarshal(ec, WithIgnoreUnused()); err != nil {
		return err
	}
	ec.Some += " is also called"
	return nil
}

type EmbeddedConfig2 struct {
	Some2 string `mapstructure:"some_2"`
}

func (ec *EmbeddedConfig2) Unmarshal(component *Conf) error {
	if err := component.Unmarshal(ec, WithIgnoreUnused()); err != nil {
		return err
	}
	ec.Some2 += " also called2"
	return nil
}

type EmbeddedConfigWithError struct {
}

func (ecwe *EmbeddedConfigWithError) Unmarshal(_ *Conf) error {
	return errors.New("embedded error")
}

type EmbeddedConfigWithMarshalError struct {
}

func (ecwe EmbeddedConfigWithMarshalError) Marshal(_ *Conf) error {
	return errors.New("marshaling error")
}

func (ecwe EmbeddedConfigWithMarshalError) Unmarshal(_ *Conf) error {
	return nil
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
	cfgMap := NewFromStringMap(map[string]any{
		"next": map[string]any{
			"string": "make sure this",
		},
		"another": "make sure this",
		"some":    "make sure this",
		"some_2":  "this better be",
	})

	tc := &testConfig{}
	assert.NoError(t, cfgMap.Unmarshal(tc))
	assert.Equal(t, "make sure this", tc.Another)
	assert.Equal(t, "make sure this is called", tc.Next.String)
	assert.Equal(t, "make sure this is also called", tc.EmbeddedConfig.Some)
	assert.Equal(t, "this better be also called2", tc.EmbeddedConfig2.Some2)
}

func TestEmbeddedUnmarshaler(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]any{
		"next": map[string]any{
			"string": "make sure this",
		},
		"another": "make sure this",
		"some":    "make sure this",
		"some_2":  "this better be",
	})

	tc := &testConfigWithoutUnmarshaler{}
	assert.NoError(t, cfgMap.Unmarshal(tc))
	assert.Equal(t, "make sure this", tc.Another)
	assert.Equal(t, "make sure this is called", tc.Next.String)
	assert.Equal(t, "make sure this is also called", tc.EmbeddedConfig.Some)
	assert.Equal(t, "this better be also called2", tc.EmbeddedConfig2.Some2)
}

func TestEmbeddedUnmarshalerError(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]any{
		"next": map[string]any{
			"string": "make sure this",
		},
		"another": "make sure this",
		"some":    "make sure this",
	})

	tc := &testConfigWithEmbeddedError{}
	assert.EqualError(t, cfgMap.Unmarshal(tc), "embedded error")
}

func TestEmbeddedMarshalerError(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]any{
		"next": map[string]any{
			"string": "make sure this",
		},
		"another": "make sure this",
	})

	tc := &testConfigWithMarshalError{}
	assert.EqualError(t, cfgMap.Unmarshal(tc), "error running encode hook: marshaling error")
}

func TestUnmarshalerKeepAlreadyInitialized(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]any{
		"next": map[string]any{
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
	cfgMap := NewFromStringMap(map[string]any{
		"next": map[string]any{
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

func (tc *errConfig) Unmarshal(*Conf) error {
	return errors.New("never works")
}

func TestUnmarshalerErr(t *testing.T) {
	cfgMap := NewFromStringMap(map[string]any{
		"err": map[string]any{
			"foo": "will not unmarshal due to error",
		},
	})

	expectErr := "1 error(s) decoding:\n\n* error decoding 'err': never works"

	tc := &testErrConfig{}
	assert.EqualError(t, cfgMap.Unmarshal(tc), expectErr)
	assert.Empty(t, tc.Err.Foo)
}

func TestZeroSliceHookFunc(t *testing.T) {
	type structWithSlices struct {
		Strings []string `mapstructure:"strings"`
	}

	tests := []struct {
		name     string
		cfg      map[string]any
		provided any
		expected any
	}{
		{
			name: "overridden by slice",
			cfg: map[string]any{
				"strings": []string{"111"},
			},
			provided: &structWithSlices{
				Strings: []string{"xxx", "yyyy", "zzzz"},
			},
			expected: &structWithSlices{
				Strings: []string{"111"},
			},
		},
		{
			name: "overridden by a bigger slice",
			cfg: map[string]any{
				"strings": []string{"111", "222", "333"},
			},
			provided: &structWithSlices{
				Strings: []string{"xxx", "yyyy"},
			},
			expected: &structWithSlices{
				Strings: []string{"111", "222", "333"},
			},
		},
		{
			name: "overridden by an empty slice",
			cfg: map[string]any{
				"strings": []string{},
			},
			provided: &structWithSlices{
				Strings: []string{"xxx", "yyyy"},
			},
			expected: &structWithSlices{
				Strings: []string{},
			},
		},
		{
			name: "not overridden by nil",
			cfg: map[string]any{
				"strings": nil,
			},
			provided: &structWithSlices{
				Strings: []string{"xxx", "yyyy"},
			},
			expected: &structWithSlices{
				Strings: []string{"xxx", "yyyy"},
			},
		},
		{
			name: "not overridden by missing value",
			cfg:  map[string]any{},
			provided: &structWithSlices{
				Strings: []string{"xxx", "yyyy"},
			},
			expected: &structWithSlices{
				Strings: []string{"xxx", "yyyy"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			cfg := NewFromStringMap(tt.cfg)

			err := cfg.Unmarshal(tt.provided)
			if assert.NoError(t, err) {
				assert.Equal(t, tt.expected, tt.provided)
			}
		})
	}
}

type C struct {
	Modifiers []string `mapstructure:"modifiers"`
}

func (c *C) Unmarshal(conf *Conf) error {
	if err := conf.Unmarshal(c); err != nil {
		return err
	}
	c.Modifiers = append(c.Modifiers, "C.Unmarshal")
	return nil
}

type B struct {
	Modifiers []string `mapstructure:"modifiers"`
	C         C        `mapstructure:"c"`
}

func (b *B) Unmarshal(conf *Conf) error {
	if err := conf.Unmarshal(b); err != nil {
		return err
	}
	b.Modifiers = append(b.Modifiers, "B.Unmarshal")
	b.C.Modifiers = append(b.C.Modifiers, "B.Unmarshal")
	return nil
}

type A struct {
	Modifiers []string `mapstructure:"modifiers"`
	B         B        `mapstructure:"b"`
}

func (a *A) Unmarshal(conf *Conf) error {
	if err := conf.Unmarshal(a); err != nil {
		return err
	}
	a.Modifiers = append(a.Modifiers, "A.Unmarshal")
	a.B.Modifiers = append(a.B.Modifiers, "A.Unmarshal")
	a.B.C.Modifiers = append(a.B.C.Modifiers, "A.Unmarshal")
	return nil
}

type Wrapper struct {
	A A `mapstructure:"a"`
}

// Test that calling the Unmarshal method on configuration structs is done from the inside out.
func TestNestedUnmarshalerImplementations(t *testing.T) {
	conf := NewFromStringMap(map[string]any{"a": map[string]any{
		"modifiers": []string{"conf.Unmarshal"},
		"b": map[string]any{
			"modifiers": []string{"conf.Unmarshal"},
			"c": map[string]any{
				"modifiers": []string{"conf.Unmarshal"},
			},
		},
	}})

	// Use a wrapper struct until we deprecate component.UnmarshalConfig
	w := &Wrapper{}
	assert.NoError(t, conf.Unmarshal(w))

	a := w.A
	assert.Equal(t, []string{"conf.Unmarshal", "A.Unmarshal"}, a.Modifiers)
	assert.Equal(t, []string{"conf.Unmarshal", "B.Unmarshal", "A.Unmarshal"}, a.B.Modifiers)
	assert.Equal(t, []string{"conf.Unmarshal", "C.Unmarshal", "B.Unmarshal", "A.Unmarshal"}, a.B.C.Modifiers)
}

// Test that unmarshaling the same conf twice works.
func TestUnmarshalDouble(t *testing.T) {
	conf := NewFromStringMap(map[string]any{
		"str": "test",
	})

	type Struct struct {
		Str string `mapstructure:"str"`
	}
	s := &Struct{}
	assert.NoError(t, conf.Unmarshal(s))
	assert.Equal(t, "test", s.Str)

	type Struct2 struct {
		Str string `mapstructure:"str"`
	}
	s2 := &Struct2{}
	assert.NoError(t, conf.Unmarshal(s2))
	assert.Equal(t, "test", s2.Str)
}

type EmbeddedStructWithUnmarshal struct {
	Foo     string `mapstructure:"foo"`
	success string
}

func (e *EmbeddedStructWithUnmarshal) Unmarshal(c *Conf) error {
	if err := c.Unmarshal(e, WithIgnoreUnused()); err != nil {
		return err
	}
	e.success = "success"
	return nil
}

type configWithUnmarshalFromEmbeddedStruct struct {
	EmbeddedStructWithUnmarshal
}

type topLevel struct {
	Cfg *configWithUnmarshalFromEmbeddedStruct `mapstructure:"toplevel"`
}

// Test that Unmarshal is called on the embedded struct on the struct.
func TestUnmarshalThroughEmbeddedStruct(t *testing.T) {
	c := NewFromStringMap(map[string]any{
		"toplevel": map[string]any{
			"foo": "bar",
		},
	})
	cfg := &topLevel{}
	err := c.Unmarshal(cfg)
	require.NoError(t, err)
	require.Equal(t, "success", cfg.Cfg.EmbeddedStructWithUnmarshal.success)
	require.Equal(t, "bar", cfg.Cfg.EmbeddedStructWithUnmarshal.Foo)
}

type configWithOwnUnmarshalAndEmbeddedSquashedStruct struct {
	EmbeddedStructWithUnmarshal `mapstructure:",squash"`
}

type topLevelSquashedEmbedded struct {
	Cfg *configWithOwnUnmarshalAndEmbeddedSquashedStruct `mapstructure:"toplevel"`
}

// Test that the Unmarshal method is called on the squashed, embedded struct.
func TestUnmarshalOwnThroughEmbeddedSquashedStruct(t *testing.T) {
	c := NewFromStringMap(map[string]any{
		"toplevel": map[string]any{
			"foo": "bar",
		},
	})
	cfg := &topLevelSquashedEmbedded{}
	err := c.Unmarshal(cfg)
	require.NoError(t, err)
	require.Equal(t, "success", cfg.Cfg.EmbeddedStructWithUnmarshal.success)
	require.Equal(t, "bar", cfg.Cfg.EmbeddedStructWithUnmarshal.Foo)
}
