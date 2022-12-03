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

package component

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type configChildStruct struct {
	Child    errConfig
	ChildPtr *errConfig
}

type configChildSlice struct {
	Child    []errConfig
	ChildPtr []*errConfig
}

type configChildMapValue struct {
	Child    map[string]errConfig
	ChildPtr map[string]*errConfig
}

type configChildMapKey struct {
	Child    map[errType]string
	ChildPtr map[*errType]string
}

type configChildTypeDef struct {
	Child    errType
	ChildPtr *errType
}

type errConfig struct {
	err error
}

func (e *errConfig) Validate() error {
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

func newErrMapType() *errMapType {
	et := errMapType(nil)
	return &et
}

func TestValidateConfig(t *testing.T) {
	var tests = []struct {
		name     string
		cfg      any
		expected error
	}{
		{
			name:     "struct",
			cfg:      errConfig{err: errors.New("struct")},
			expected: errors.New("struct"),
		},
		{
			name:     "pointer struct",
			cfg:      &errConfig{err: errors.New("pointer struct")},
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
			cfg:      configChildStruct{Child: errConfig{err: errors.New("child struct")}},
			expected: errors.New("child struct"),
		},
		{
			name:     "pointer child struct",
			cfg:      &configChildStruct{Child: errConfig{err: errors.New("pointer child struct")}},
			expected: errors.New("pointer child struct"),
		},
		{
			name:     "child struct pointer",
			cfg:      &configChildStruct{ChildPtr: &errConfig{err: errors.New("child struct pointer")}},
			expected: errors.New("child struct pointer"),
		},
		{
			name:     "child slice",
			cfg:      configChildSlice{Child: []errConfig{{}, {err: errors.New("child slice")}}},
			expected: errors.New("child slice"),
		},
		{
			name:     "pointer child slice",
			cfg:      &configChildSlice{Child: []errConfig{{}, {err: errors.New("pointer child slice")}}},
			expected: errors.New("pointer child slice"),
		},
		{
			name:     "child slice pointer",
			cfg:      &configChildSlice{ChildPtr: []*errConfig{{}, {err: errors.New("child slice pointer")}}},
			expected: errors.New("child slice pointer"),
		},
		{
			name:     "child map value",
			cfg:      configChildMapValue{Child: map[string]errConfig{"test": {err: errors.New("child map")}}},
			expected: errors.New("child map"),
		},
		{
			name:     "pointer child map value",
			cfg:      &configChildMapValue{Child: map[string]errConfig{"test": {err: errors.New("pointer child map")}}},
			expected: errors.New("pointer child map"),
		},
		{
			name:     "child map value pointer",
			cfg:      &configChildMapValue{ChildPtr: map[string]*errConfig{"test": {err: errors.New("child map pointer")}}},
			expected: errors.New("child map pointer"),
		},
		{
			name:     "child map key",
			cfg:      configChildMapKey{Child: map[errType]string{"child map key": ""}},
			expected: errors.New("child map key"),
		},
		{
			name:     "pointer child map key",
			cfg:      &configChildMapKey{Child: map[errType]string{"pointer child map key": ""}},
			expected: errors.New("pointer child map key"),
		},
		{
			name:     "child map key pointer",
			cfg:      &configChildMapKey{ChildPtr: map[*errType]string{newErrType("child map key pointer"): ""}},
			expected: errors.New("child map key pointer"),
		},
		{
			name:     "child type",
			cfg:      configChildTypeDef{Child: "child type"},
			expected: errors.New("child type"),
		},
		{
			name:     "pointer child type",
			cfg:      &configChildTypeDef{Child: "pointer child type"},
			expected: errors.New("pointer child type"),
		},
		{
			name:     "child type pointer",
			cfg:      &configChildTypeDef{ChildPtr: newErrType("child type pointer")},
			expected: errors.New("child type pointer"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(reflect.ValueOf(tt.cfg))
			assert.Equal(t, tt.expected, err)
		})
	}
}
