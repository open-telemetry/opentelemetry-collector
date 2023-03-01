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
	parent SliceParent[*[]otlpcommon.AnyValue]
}

type stubSliceParent struct {
	orig *[]otlpcommon.AnyValue
}

func (mp stubSliceParent) EnsureMutability() {}

func (mp stubSliceParent) GetChildOrig() *[]otlpcommon.AnyValue {
	return mp.orig
}

func (mp stubSliceParent) GetState() *State {
	state := StateExclusive
	return &state
}

var _ SliceParent[*[]otlpcommon.AnyValue] = (*stubSliceParent)(nil)

func (ms Slice) GetOrig() *[]otlpcommon.AnyValue {
	return ms.parent.GetChildOrig()
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
	ms.parent.EnsureMutability()
}

func (ms Slice) GetState() *State {
	return ms.parent.GetState()
}

func NewSliceFromOrig(orig *[]otlpcommon.AnyValue) Slice {
	return Slice{&stubSliceParent{orig: orig}}
}

func NewSliceFromParent(parent SliceParent[*[]otlpcommon.AnyValue]) Slice {
	return Slice{parent: parent}
}

func GenerateTestSlice() Slice {
	orig := []otlpcommon.AnyValue{}
	tv := NewSliceFromOrig(&orig)
	FillTestSlice(tv)
	return tv
}

func FillTestSlice(tv Slice) {
	*tv.GetOrig() = make([]otlpcommon.AnyValue, 7)
	for i := 0; i < 7; i++ {
		FillTestValue(NewValue(&(*tv.GetOrig())[i], nil))
	}
}
