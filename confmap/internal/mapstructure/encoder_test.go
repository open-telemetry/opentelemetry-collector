// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package mapstructure

import (
	"encoding"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/require"
)

type TestComplexStruct struct {
	Skipped               TestEmptyStruct             `mapstructure:",squash"`
	Nested                TestSimpleStruct            `mapstructure:",squash"`
	Slice                 []TestSimpleStruct          `mapstructure:"slice,omitempty"`
	Pointer               *TestSimpleStruct           `mapstructure:"ptr"`
	Map                   map[string]TestSimpleStruct `mapstructure:"map,omitempty"`
	Remain                map[string]any              `mapstructure:",remain"`
	TranslatedYaml        TestYamlStruct              `mapstructure:"translated"`
	SquashedYaml          TestYamlStruct              `mapstructure:",squash"`
	PointerTranslatedYaml *TestPtrToYamlStruct        `mapstructure:"translated_ptr"`
	PointerSquashedYaml   *TestPtrToYamlStruct        `mapstructure:",squash"`
	Interface             encoding.TextMarshaler
}

type TestSimpleStruct struct {
	Value   string `mapstructure:"value"`
	skipped string
	err     error
}

type TestEmptyStruct struct {
	Value string `mapstructure:"-"`
}

type TestYamlStruct struct {
	YamlValue     string               `yaml:"yaml_value"`
	YamlOmitEmpty string               `yaml:"yaml_omit,omitempty"`
	YamlInline    TestYamlSimpleStruct `yaml:",inline"`
}

type TestPtrToYamlStruct struct {
	YamlValue     string                     `yaml:"yaml_value_ptr"`
	YamlOmitEmpty string                     `yaml:"yaml_omit_ptr,omitempty"`
	YamlInline    *TestYamlPtrToSimpleStruct `yaml:",inline"`
}

type TestYamlSimpleStruct struct {
	Inline string `yaml:"yaml_inline"`
}

type TestYamlPtrToSimpleStruct struct {
	InlinePtr string `yaml:"yaml_inline_ptr"`
}

type TestID string

func (tID TestID) MarshalText() (text []byte, err error) {
	out := string(tID)
	if out == "error" {
		return nil, errors.New("parsing error")
	}
	if !strings.HasSuffix(out, "_") {
		out += "_"
	}
	return []byte(out), nil
}

type TestStringLike string

func TestEncode(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: mapstructure.ComposeDecodeHookFunc(
			YamlMarshalerHookFunc(),
			TextMarshalerHookFunc(),
		),
	})
	testCases := map[string]struct {
		input any
		want  any
	}{
		"WithString": {
			input: "test",
			want:  "test",
		},
		"WithTextMarshaler": {
			input: TestID("type"),
			want:  "type_",
		},
		"MapWithTextMarshalerKey": {
			input: map[TestID]TestSimpleStruct{
				TestID("type"): {Value: "value"},
			},
			want: map[string]any{
				"type_": map[string]any{"value": "value"},
			},
		},
		"MapWithoutTextMarshalerKey": {
			input: map[TestStringLike]TestSimpleStruct{
				TestStringLike("key"): {Value: "value"},
			},
			want: map[string]any{
				"key": map[string]any{"value": "value"},
			},
		},
		"WithSlice": {
			input: []TestID{
				TestID("nop"),
				TestID("type_"),
			},
			want: []any{"nop_", "type_"},
		},
		"WithSimpleStruct": {
			input: TestSimpleStruct{Value: "test", skipped: "skipped"},
			want: map[string]any{
				"value": "test",
			},
		},
		"WithComplexStruct": {
			input: &TestComplexStruct{
				Skipped: TestEmptyStruct{
					Value: "omitted",
				},
				Nested: TestSimpleStruct{
					Value: "nested",
				},
				Slice: []TestSimpleStruct{
					{Value: "slice"},
				},
				Map: map[string]TestSimpleStruct{
					"Key": {Value: "map"},
				},
				Pointer: &TestSimpleStruct{
					Value: "pointer",
				},
				Remain: map[string]any{
					"remain1": 23,
					"remain2": "value",
				},
				Interface: TestID("value"),
				TranslatedYaml: TestYamlStruct{
					YamlValue:     "foo_translated",
					YamlOmitEmpty: "",
					YamlInline: TestYamlSimpleStruct{
						Inline: "bar_translated",
					},
				},
				SquashedYaml: TestYamlStruct{
					YamlValue:     "foo_squashed",
					YamlOmitEmpty: "",
					YamlInline: TestYamlSimpleStruct{
						Inline: "bar_squashed",
					},
				},
				PointerTranslatedYaml: &TestPtrToYamlStruct{
					YamlValue:     "foo_translated_ptr",
					YamlOmitEmpty: "",
					YamlInline: &TestYamlPtrToSimpleStruct{
						InlinePtr: "bar_translated_ptr",
					},
				},
				PointerSquashedYaml: &TestPtrToYamlStruct{
					YamlValue:     "foo_squashed_ptr",
					YamlOmitEmpty: "",
					YamlInline: &TestYamlPtrToSimpleStruct{
						InlinePtr: "bar_squashed_ptr",
					},
				},
			},
			want: map[string]any{
				"value": "nested",
				"slice": []any{map[string]any{"value": "slice"}},
				"map": map[string]any{
					"Key": map[string]any{"value": "map"},
				},
				"ptr":         map[string]any{"value": "pointer"},
				"interface":   "value_",
				"yaml_value":  "foo_squashed",
				"yaml_inline": "bar_squashed",
				"translated": map[string]any{
					"yaml_value":  "foo_translated",
					"yaml_inline": "bar_translated",
				},
				"yaml_value_ptr":  "foo_squashed_ptr",
				"yaml_inline_ptr": "bar_squashed_ptr",
				"translated_ptr": map[string]any{
					"yaml_value_ptr":  "foo_translated_ptr",
					"yaml_inline_ptr": "bar_translated_ptr",
				},
				"remain1": 23,
				"remain2": "value",
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := enc.Encode(testCase.input)
			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}
	// without the TextMarshalerHookFunc
	enc.config.EncodeHook = nil
	testCase := TestID("test")
	got, err := enc.Encode(testCase)
	require.NoError(t, err)
	require.Equal(t, testCase, got)
}

func TestGetTagInfo(t *testing.T) {
	testCases := []struct {
		name       string
		field      reflect.StructField
		wantName   string
		wantOmit   bool
		wantSquash bool
	}{
		{
			name: "WithoutTags",
			field: reflect.StructField{
				Name: "Test",
			},
			wantName: "test",
		},
		{
			name: "WithoutMapStructureTag",
			field: reflect.StructField{
				Tag:  `yaml:"hello,inline"`,
				Name: "YAML",
			},
			wantName: "yaml",
		},
		{
			name: "WithRename",
			field: reflect.StructField{
				Tag:  `mapstructure:"hello"`,
				Name: "Test",
			},
			wantName: "hello",
		},
		{
			name: "WithOmitEmpty",
			field: reflect.StructField{
				Tag:  `mapstructure:"hello,omitempty"`,
				Name: "Test",
			},
			wantName: "hello",
			wantOmit: true,
		},
		{
			name: "WithSquash",
			field: reflect.StructField{
				Tag:  `mapstructure:",squash"`,
				Name: "Test",
			},
			wantSquash: true,
		},
		{
			name: "WithRemain",
			field: reflect.StructField{
				Tag:  `mapstructure:",remain"`,
				Name: "Test",
			},
			wantSquash: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := getTagInfo(tt.field)
			require.Equal(t, tt.wantName, got.name)
			require.Equal(t, tt.wantOmit, got.omitEmpty)
			require.Equal(t, tt.wantSquash, got.squash)
		})
	}
}

func TestEncodeValueError(t *testing.T) {
	enc := New(nil)
	testValue := reflect.ValueOf("")
	testCases := []struct {
		encodeFn func(value reflect.Value) (any, error)
		wantErr  error
	}{
		{encodeFn: enc.encodeMap, wantErr: &reflect.ValueError{Method: "encodeMap", Kind: reflect.String}},
		{encodeFn: enc.encodeStruct, wantErr: &reflect.ValueError{Method: "encodeStruct", Kind: reflect.String}},
		{encodeFn: enc.encodeSlice, wantErr: &reflect.ValueError{Method: "encodeSlice", Kind: reflect.String}},
	}
	for _, tt := range testCases {
		got, err := tt.encodeFn(testValue)
		require.Error(t, err)
		require.Equal(t, tt.wantErr, err)
		require.Nil(t, got)
	}
}

func TestEncodeNonStringEncodedKey(t *testing.T) {
	enc := New(nil)
	testCase := []struct {
		Test map[string]any
	}{
		{
			Test: map[string]any{
				"test": map[TestEmptyStruct]TestSimpleStruct{
					{Value: "key"}: {Value: "value"},
				},
			},
		},
	}
	got, err := enc.Encode(testCase)
	require.Error(t, err)
	require.ErrorIs(t, err, errNonStringEncodedKey)
	require.Nil(t, got)
}

func TestDuplicateKey(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: TextMarshalerHookFunc(),
	})
	testCase := map[TestID]string{
		"test":  "value",
		"test_": "other value",
	}
	got, err := enc.Encode(testCase)
	require.Error(t, err)
	require.Nil(t, got)
}

func TestTextMarshalerError(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: TextMarshalerHookFunc(),
	})
	testCase := map[TestID]string{
		"error": "value",
	}
	got, err := enc.Encode(testCase)
	require.Error(t, err)
	require.Nil(t, got)
}

func TestEncodeStruct(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: testHookFunc(),
	})
	testCase := TestSimpleStruct{
		Value:   "original",
		skipped: "final",
	}
	got, err := enc.Encode(testCase)
	require.NoError(t, err)
	require.Equal(t, "final", got)
}

func TestEncodeStructError(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: testHookFunc(),
	})
	wantErr := errors.New("test")
	testCase := map[TestSimpleStruct]string{
		{err: wantErr}: "value",
	}
	got, err := enc.Encode(testCase)
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
}

func TestEncodeNil(t *testing.T) {
	enc := New(nil)
	got, err := enc.Encode(nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

func testHookFunc() mapstructure.DecodeHookFuncValue {
	return func(from, _ reflect.Value) (any, error) {
		if from.Kind() != reflect.Struct {
			return from.Interface(), nil
		}

		got, ok := from.Interface().(TestSimpleStruct)
		if !ok {
			return from.Interface(), nil
		}
		if got.err != nil {
			return nil, got.err
		}
		return got.skipped, nil
	}
}
