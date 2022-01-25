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

package pdata // import "go.opentelemetry.io/collector/model/pdata"

import (
	"go.opentelemetry.io/collector/model/internal/data"
)

// OptionalDouble is an alias of OTLP OptionalDouble data type.
type OptionalDouble struct {
	orig data.OptionalDouble
}

// NewOptionalDouble returns a new OptionalDouble from the given byte array.
func NewOptionalDouble(value float64) OptionalDouble {
	return OptionalDouble{orig: data.NewOptionalDouble(value)}
}

// IsEmpty returns true if id doesn't contain at least one non-zero byte.
func (t OptionalDouble) IsEmpty() bool {
	return t.orig.IsEmpty()
}

func (t OptionalDouble) Value() float64 {
	return t.orig.DoubleValue
}
