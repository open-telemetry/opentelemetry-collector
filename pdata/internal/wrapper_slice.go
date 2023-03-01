// Copyright The OpenTelemetry Authors
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

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type Slice struct {
	*pSlice
}

type pSlice struct {
	orig   *[]otlpcommon.AnyValue
	state  *State
	parent Parent[*[]otlpcommon.AnyValue]
}

func (ms Slice) GetOrig() *[]otlpcommon.AnyValue {
	return ms.orig
}

func (ms Slice) GetValueParent(idx int) SliceValueParent {
	return SliceValueParent{Slice: ms, idx: idx}
}

type SliceValueParent struct {
	Slice
	idx int
}

func (ps SliceValueParent) RefreshOrigState() (*otlpcommon.AnyValue, *State) {
	return &(*ps.GetOrig())[ps.idx], ps.GetState()
}

func (ms Slice) EnsureMutability() {
	if *ms.state == StateShared {
		ms.parent.EnsureMutability()
	}
}

func (ms Slice) GetState() *State {
	return ms.state
}

func NewSlice(orig *[]otlpcommon.AnyValue, parent Parent[*[]otlpcommon.AnyValue]) Slice {
	if parent == nil {
		state := StateExclusive
		return Slice{&pSlice{orig: orig, state: &state}}
	}
	return Slice{&pSlice{orig: orig, state: parent.GetState(), parent: parent}}
}

func GenerateTestSlice() Slice {
	orig := []otlpcommon.AnyValue{}
	tv := NewSlice(&orig, nil)
	FillTestSlice(tv)
	return tv
}

func FillTestSlice(tv Slice) {
	*tv.orig = make([]otlpcommon.AnyValue, 7)
	for i := 0; i < 7; i++ {
		FillTestValue(NewValue(&(*tv.orig)[i], nil))
	}
}
