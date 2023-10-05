// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package mapstructure

import (
	"encoding"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

type TestComplexStruct struct {
	Skipped   TestEmptyStruct             `mapstructure:",squash"`
	Nested    TestSimpleStruct            `mapstructure:",squash"`
	Slice     []TestSimpleStruct          `mapstructure:"slice,omitempty"`
	Array     [2]string                   `mapstructure:"array,omitempty"`
	Pointer   *TestSimpleStruct           `mapstructure:"ptr"`
	Map       map[string]TestSimpleStruct `mapstructure:"map,omitempty"`
	Remain    map[string]any              `mapstructure:",remain"`
	Interface encoding.TextMarshaler
	Function  func() string
	Channel   chan string
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
		EncodeHook: mapstructure.ComposeDecodeHookFunc(
			TextMarshalerHookFunc(),
			UnsupportedKindHookFunc(),
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
				Array: [2]string{"one", "two"},
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
				Function: func() string {
					return "ignore"
				},
				Channel: make(chan string),
			},
			want: map[string]any{
				"value": "nested",
				"slice": []any{map[string]any{"value": "slice"}},
				"array": []any{"one", "two"},
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
			assert.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
	// without the TextMarshalerHookFunc
	enc.config.EncodeHook = nil
	testCase := TestID("test")
	got, err := enc.Encode(testCase)
	assert.NoError(t, err)
	assert.Equal(t, testCase, got)
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
			assert.Equal(t, testCase.wantName, got.name)
			assert.Equal(t, testCase.wantOmit, got.omitEmpty)
			assert.Equal(t, testCase.wantSquash, got.squash)
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
		assert.Error(t, err)
		assert.Equal(t, testCase.wantErr, err)
		assert.Nil(t, got)
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
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errNonStringEncodedKey))
	assert.Nil(t, got)
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
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTextMarshalerError(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: TextMarshalerHookFunc(),
	})
	testCase := map[TestID]string{
		"error": "value",
	}
	got, err := enc.Encode(testCase)
	assert.Error(t, err)
	assert.Nil(t, got)
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
	assert.NoError(t, err)
	assert.Equal(t, "final", got)
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
	assert.Error(t, err)
	assert.True(t, errors.Is(err, wantErr))
	assert.Nil(t, got)
}

func TestEncodeNil(t *testing.T) {
	enc := New(nil)
	got, err := enc.Encode(nil)
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestUnsupportedKind(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: UnsupportedKindHookFunc(),
	})
	testCases := map[reflect.Kind]any{
		reflect.Func: func() string {
			return "unsupported"
		},
		reflect.Chan: make(chan string),
	}
	for kind, input := range testCases {
		t.Run(kind.String(), func(t *testing.T) {
			got, err := enc.Encode(input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, errUnsupportedKind)
			assert.Nil(t, got)
		})
	}
}

type TestFunction func() string

func (tf TestFunction) MarshalText() (text []byte, err error) {
	return []byte(tf()), nil
}

func TestUnsupportedKindWithMarshaler(t *testing.T) {
	enc := New(&EncoderConfig{
		EncodeHook: mapstructure.ComposeDecodeHookFunc(
			TextMarshalerHookFunc(),
			UnsupportedKindHookFunc(),
		),
	})
	testCase := TestFunction(func() string {
		return "marshal"
	})
	got, err := enc.Encode(testCase)
	assert.NoError(t, err)
	assert.Equal(t, "marshal", got)
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
