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

package json // import "go.opentelemetry.io/collector/pdata/internal/json"

import (
	jsoniter "github.com/json-iterator/go"
)

// ReadInt64  Unmarshal JSON data and return int64
func ReadInt64(iter *jsoniter.Iterator) int64 {
	return iter.ReadAny().ToInt64()
}
