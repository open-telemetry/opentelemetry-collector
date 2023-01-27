// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mapstructure

import (
	"encoding"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

type TestComplexStruct struct {
	Skipped   TestEmptyStruct             `mapstructure:",squash"`
	Nested    TestSimpleStruct            `mapstructure:",squash"`
	Slice     []TestSimpleStruct          `mapstructure:"slice,omitempty"`
	Pointer   *TestSimpleStruct           `mapstructure:"ptr"`
	Map       map[string]TestSimpleStruct `mapstructure:"map,omitempty"`
	Remain    map[string]any              `mapstructure:",remain"`
	Interface encoding.TextMarshaler
}

type TestSimpleStruct struct {
	Value   string `mapstructure:"value"`
	skipped string
	err     error
}

type TestEmptyStruct struct {
	Value string `mapstructure:"-"`
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

func TestEncode(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: TextMarshalerHookFunc(),
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
			},
			want: map[string]any{
				"value": "nested",
				"slice": []any{map[string]any{"value": "slice"}},
				"map": map[string]any{
					"Key": map[string]any{"value": "map"},
				},
				"ptr":       map[string]any{"value": "pointer"},
				"interface": "value_",
				"remain1":   23,
				"remain2":   "value",
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
	testCases := map[string]struct {
		field      reflect.StructField
		wantName   string
		wantOmit   bool
		wantSquash bool
	}{
		"WithoutTags": {
			field: reflect.StructField{
				Name: "Test",
			},
			wantName: "test",
		},
		"WithoutMapStructureTag": {
			field: reflect.StructField{
				Tag:  `yaml:"hello,inline"`,
				Name: "YAML",
			},
			wantName: "yaml",
		},
		"WithRename": {
			field: reflect.StructField{
				Tag:  `mapstructure:"hello"`,
				Name: "Test",
			},
			wantName: "hello",
		},
		"WithOmitEmpty": {
			field: reflect.StructField{
				Tag:  `mapstructure:"hello,omitempty"`,
				Name: "Test",
			},
			wantName: "hello",
			wantOmit: true,
		},
		"WithSquash": {
			field: reflect.StructField{
				Tag:  `mapstructure:",squash"`,
				Name: "Test",
			},
			wantSquash: true,
		},
		"WithRemain": {
			field: reflect.StructField{
				Tag:  `mapstructure:",remain"`,
				Name: "Test",
			},
			wantSquash: true,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			got := getTagInfo(testCase.field)
			require.Equal(t, testCase.wantName, got.name)
			require.Equal(t, testCase.wantOmit, got.omitEmpty)
			require.Equal(t, testCase.wantSquash, got.squash)
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
	for _, testCase := range testCases {
		got, err := testCase.encodeFn(testValue)
		require.Error(t, err)
		require.Equal(t, testCase.wantErr, err)
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
	require.True(t, errors.Is(err, errNonStringEncodedKey))
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
	require.True(t, errors.Is(err, wantErr))
	require.Nil(t, got)
}

func TestEncodeNil(t *testing.T) {
	enc := New(nil)
	got, err := enc.Encode(nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

func testHookFunc() mapstructure.DecodeHookFuncValue {
	return func(from reflect.Value, _ reflect.Value) (any, error) {
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
