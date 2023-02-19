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

type Map struct {
	orig *[]otlpcommon.KeyValue
}

func GetOrigMap(ms Map) *[]otlpcommon.KeyValue {
	return ms.orig
}

func NewMap(orig *[]otlpcommon.KeyValue) Map {
	return Map{orig: orig}
}

func CopyOrigMap(dst, src *[]otlpcommon.KeyValue) {
	newLen := len(*src)
	oldCap := cap(*dst)
	if newLen <= oldCap {
		// New slice fits in existing slice, no need to reallocate.
		*dst = (*dst)[:newLen:oldCap]
		for i := range *src {
			akv := &(*src)[i]
			destAkv := &(*dst)[i]
			destAkv.Key = akv.Key
			CopyOrigValue(&destAkv.Value, &akv.Value)
		}
		return
	}

	// New slice is bigger than exist slice. Allocate new space.
	origs := make([]otlpcommon.KeyValue, len(*src))
	for i := range *src {
		akv := &(*src)[i]
		origs[i].Key = akv.Key
		CopyOrigValue(&origs[i].Value, &akv.Value)
	}
	*dst = origs
}

func GenerateTestMap() Map {
	var orig []otlpcommon.KeyValue
	ms := NewMap(&orig)
	FillTestMap(ms)
	return ms
}

func FillTestMap(dest Map) {
	*dest.orig = nil
	*dest.orig = append(*dest.orig, otlpcommon.KeyValue{Key: "k", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}}})
}
