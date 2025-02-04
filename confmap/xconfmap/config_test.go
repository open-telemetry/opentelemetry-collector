// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type configChildStruct struct {
	Child    errValidateConfig
	ChildPtr *errValidateConfig
}

type configChildSlice struct {
	Child    []errValidateConfig
	ChildPtr []*errValidateConfig
}

type configChildMapValue struct {
	Child    map[string]errValidateConfig
	ChildPtr map[string]*errValidateConfig
}

type configChildMapKey struct {
	Child    map[errType]string
	ChildPtr map[*errType]string
}

type configChildTypeDef struct {
	Child    errType
	ChildPtr *errType
}

type config any

type configChildInterface struct {
	Child config
}

type errValidateConfig struct {
	err error
}

func (e *errValidateConfig) Validate() error {
	return e.err
}

type errType string

func (e errType) Validate() error {
	if e == "" {
		return nil
	}
	return errors.New(string(e))
}

func newErrType(etStr string) *errType {
	et := errType(etStr)
	return &et
}

type errMapType map[string]string

func (e errMapType) Validate() error {
	return errors.New(e["err"])
}

type structKey struct {
	k string
	e error
}

func (s structKey) String() string {
	return s.k
}

func (s structKey) Validate() error {
	return s.e
}

type configChildMapCustomKey struct {
	Child map[structKey]errValidateConfig
}

func newErrMapType() *errMapType {
	et := errMapType(nil)
	return &et
}

type configMapstructure struct {
	Valid  *errValidateConfig `mapstructure:"validtag,omitempty"`
	NoData *errValidateConfig `mapstructure:""`
	NoName *errValidateConfig `mapstructure:",remain"`
}

type configDeeplyNested struct {
	MapKeyChild   map[configChildStruct]string
	MapValueChild map[string]configChildStruct
	SliceChild    []configChildSlice
	MapIntKey     map[int]errValidateConfig
	MapFloatKey   map[float64]errValidateConfig
}

type sliceTypeAlias []configChildSlice

func (sliceTypeAlias) Validate() error {
	return errors.New("sliceTypeAlias error")
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      any
		expected error
	}{
		{
			name:     "struct",
			cfg:      errValidateConfig{err: errors.New("struct")},
			expected: errors.New("struct"),
		},
		{
			name:     "pointer struct",
			cfg:      &errValidateConfig{err: errors.New("pointer struct")},
			expected: errors.New("pointer struct"),
		},
		{
			name:     "type",
			cfg:      errType("type"),
			expected: errors.New("type"),
		},
		{
			name:     "pointer child",
			cfg:      newErrType("pointer type"),
			expected: errors.New("pointer type"),
		},
		{
			name:     "child interface with nil",
			cfg:      configChildInterface{},
			expected: nil,
		},
		{
			name:     "pointer to child interface with nil",
			cfg:      &configChildInterface{},
			expected: nil,
		},
		{
			name:     "nil",
			cfg:      nil,
			expected: nil,
		},
		{
			name:     "nil map type",
			cfg:      errMapType(nil),
			expected: errors.New(""),
		},
		{
			name:     "nil pointer map type",
			cfg:      newErrMapType(),
			expected: errors.New(""),
		},
		{
			name:     "child struct",
			cfg:      configChildStruct{Child: errValidateConfig{err: errors.New("child struct")}},
			expected: errors.New("child: child struct"),
		},
		{
			name:     "pointer child struct",
			cfg:      &configChildStruct{Child: errValidateConfig{err: errors.New("pointer child struct")}},
			expected: errors.New("child: pointer child struct"),
		},
		{
			name:     "child struct pointer",
			cfg:      &configChildStruct{ChildPtr: &errValidateConfig{err: errors.New("child struct pointer")}},
			expected: errors.New("childptr: child struct pointer"),
		},
		{
			name:     "child interface",
			cfg:      configChildInterface{Child: errValidateConfig{err: errors.New("child interface")}},
			expected: errors.New("child: child interface"),
		},
		{
			name:     "pointer to child interface",
			cfg:      &configChildInterface{Child: errValidateConfig{err: errors.New("pointer to child interface")}},
			expected: errors.New("child: pointer to child interface"),
		},
		{
			name:     "child interface with pointer",
			cfg:      configChildInterface{Child: &errValidateConfig{err: errors.New("child interface with pointer")}},
			expected: errors.New("child: child interface with pointer"),
		},
		{
			name:     "pointer to child interface with pointer",
			cfg:      &configChildInterface{Child: &errValidateConfig{err: errors.New("pointer to child interface with pointer")}},
			expected: errors.New("child: pointer to child interface with pointer"),
		},
		{
			name:     "child slice",
			cfg:      configChildSlice{Child: []errValidateConfig{{}, {err: errors.New("child slice")}}},
			expected: errors.New("child::1: child slice"),
		},
		{
			name:     "pointer child slice",
			cfg:      &configChildSlice{Child: []errValidateConfig{{}, {err: errors.New("pointer child slice")}}},
			expected: errors.New("child::1: pointer child slice"),
		},
		{
			name:     "child slice pointer",
			cfg:      &configChildSlice{ChildPtr: []*errValidateConfig{{}, {err: errors.New("child slice pointer")}}},
			expected: errors.New("childptr::1: child slice pointer"),
		},
		{
			name:     "child map value",
			cfg:      configChildMapValue{Child: map[string]errValidateConfig{"test": {err: errors.New("child map")}}},
			expected: errors.New("child::test: child map"),
		},
		{
			name:     "pointer child map value",
			cfg:      &configChildMapValue{Child: map[string]errValidateConfig{"test": {err: errors.New("pointer child map")}}},
			expected: errors.New("child::test: pointer child map"),
		},
		{
			name:     "child map value pointer",
			cfg:      &configChildMapValue{ChildPtr: map[string]*errValidateConfig{"test": {err: errors.New("child map pointer")}}},
			expected: errors.New("childptr::test: child map pointer"),
		},
		{
			name:     "child map key",
			cfg:      configChildMapKey{Child: map[errType]string{"child_map_key": ""}},
			expected: errors.New("child::child_map_key: child_map_key"),
		},
		{
			name:     "pointer child map key",
			cfg:      &configChildMapKey{Child: map[errType]string{"pointer_child_map_key": ""}},
			expected: errors.New("child::pointer_child_map_key: pointer_child_map_key"),
		},
		{
			name:     "child map key pointer",
			cfg:      &configChildMapKey{ChildPtr: map[*errType]string{newErrType("child map key pointer"): ""}},
			expected: errors.New("childptr::[*xconfmap.errType key]: child map key pointer"),
		},
		{
			name:     "map with stringified non-string key type",
			cfg:      &configChildMapCustomKey{Child: map[structKey]errValidateConfig{{k: "struct_key", e: errors.New("custom key error")}: {err: errors.New("value error")}}},
			expected: errors.New("child::struct_key: custom key error\nchild::struct_key: value error"),
		},
		{
			name:     "child type",
			cfg:      configChildTypeDef{Child: "child type"},
			expected: errors.New("child: child type"),
		},
		{
			name:     "pointer child type",
			cfg:      &configChildTypeDef{Child: "pointer child type"},
			expected: errors.New("child: pointer child type"),
		},
		{
			name:     "child type pointer",
			cfg:      &configChildTypeDef{ChildPtr: newErrType("child type pointer")},
			expected: errors.New("childptr: child type pointer"),
		},
		{
			name:     "valid mapstructure tag",
			cfg:      configMapstructure{Valid: &errValidateConfig{errors.New("test")}},
			expected: errors.New("validtag: test"),
		},
		{
			name:     "zero-length mapstructure tag",
			cfg:      configMapstructure{NoData: &errValidateConfig{errors.New("test")}},
			expected: errors.New("nodata: test"),
		},
		{
			name:     "no field name in mapstructure tag",
			cfg:      configMapstructure{NoName: &errValidateConfig{errors.New("test")}},
			expected: errors.New("noname: test"),
		},
		{
			name:     "nested map key error",
			cfg:      configDeeplyNested{MapKeyChild: map[configChildStruct]string{{Child: errValidateConfig{err: errors.New("child key error")}}: "val"}},
			expected: errors.New("mapkeychild::[xconfmap.configChildStruct key]::child: child key error"),
		},
		{
			name:     "nested map value error",
			cfg:      configDeeplyNested{MapValueChild: map[string]configChildStruct{"key": {Child: errValidateConfig{err: errors.New("child key error")}}}},
			expected: errors.New("mapvaluechild::key::child: child key error"),
		},
		{
			name:     "nested slice value error",
			cfg:      configDeeplyNested{SliceChild: []configChildSlice{{Child: []errValidateConfig{{err: errors.New("child key error")}}}}},
			expected: errors.New("slicechild::0::child::0: child key error"),
		},
		{
			name:     "nested map with int key",
			cfg:      configDeeplyNested{MapIntKey: map[int]errValidateConfig{1: {err: errors.New("int key error")}}},
			expected: errors.New("mapintkey::1: int key error"),
		},
		{
			name:     "nested map with float key",
			cfg:      configDeeplyNested{MapFloatKey: map[float64]errValidateConfig{1.2: {err: errors.New("float key error")}}},
			expected: errors.New("mapfloatkey::1.2: float key error"),
		},
		{
			name:     "slice type alias",
			cfg:      sliceTypeAlias{},
			expected: errors.New("sliceTypeAlias error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)

			if tt.expected != nil {
				assert.EqualError(t, err, tt.expected.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
