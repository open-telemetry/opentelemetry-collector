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

// NumberDataPointValueType specifies the type of NumberDataPoint value.
type NumberDataPointValueType int32

const (
	// NumberDataPointValueTypeEmpty means that data point value is unset.
	NumberDataPointValueTypeEmpty NumberDataPointValueType = iota
	NumberDataPointValueTypeInt
	NumberDataPointValueTypeDouble
)

// String returns the string representation of the NumberDataPointValueType.
func (nt NumberDataPointValueType) String() string {
	switch nt {
	case NumberDataPointValueTypeEmpty:
		return "Empty"
	case NumberDataPointValueTypeInt:
		return "Int"
	case NumberDataPointValueTypeDouble:
		return "Double"
	}
	return ""
}
