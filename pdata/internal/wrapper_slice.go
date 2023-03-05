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
	parent Parent[*[]otlpcommon.AnyValue]
}

type stubSliceParent struct {
	orig *[]otlpcommon.AnyValue
}

func (sp stubSliceParent) EnsureMutability() {}

func (sp stubSliceParent) GetChildOrig() *[]otlpcommon.AnyValue {
	return sp.orig
}

var _ Parent[*[]otlpcommon.AnyValue] = (*stubSliceParent)(nil)

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

func (ps SliceValueParent) GetChildOrig() *otlpcommon.AnyValue {
	return &(*ps.GetOrig())[ps.idx]
}

func (ms Slice) EnsureMutability() {
	ms.parent.EnsureMutability()
}

func NewSliceFromOrig(orig *[]otlpcommon.AnyValue) Slice {
	return Slice{parent: &stubSliceParent{orig: orig}}
}

func NewSliceFromParent(parent Parent[*[]otlpcommon.AnyValue]) Slice {
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
		FillTestValue(NewValueFromOrig(&(*tv.GetOrig())[i]))
	}
}
