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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

// ExemplarValueType specifies the type of Exemplar measurement value.
type ExemplarValueType int32

const (
	// ExemplarValueTypeEmpty means that exemplar value is unset.
	ExemplarValueTypeEmpty ExemplarValueType = iota
	ExemplarValueTypeInt
	ExemplarValueTypeDouble
)

// String returns the string representation of the ExemplarValueType.
func (nt ExemplarValueType) String() string {
	switch nt {
	case ExemplarValueTypeEmpty:
		return "Empty"
	case ExemplarValueTypeInt:
		return "Int"
	case ExemplarValueTypeDouble:
		return "Double"
	}
	return ""
}
