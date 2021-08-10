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

package pdata

import (
	"go.opentelemetry.io/collector/model/internal/data"
)

type PDataContext struct {
	orig *data.PDataContext
}

func newPDataContext(orig *data.PDataContext) PDataContext {
	return PDataContext{orig}
}

func (pdc PDataContext) Set(key, val interface{}) {
	pdc.orig.List = append(pdc.orig.List, data.PDataContextKeyValue{Key: key, Value: val})
}

func (pdc PDataContext) Get(key interface{}) interface{} {
	for _, kv := range pdc.orig.List {
		if kv.Key == key {
			return kv.Value
		}
	}
	return nil
}

func (pdc PDataContext) Range(f func(k interface{}, v interface{})) {
	for _, kv := range pdc.orig.List {
		f(kv.Key, kv.Value)
	}
}
