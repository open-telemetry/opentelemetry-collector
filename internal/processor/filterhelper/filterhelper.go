// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterhelper

import (
	"fmt"

	"github.com/spf13/cast"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

// NilAttributeValue is used to convert the raw `value` from ActionKeyValue to the supported trace attribute values.
func NewAttributeValue(value interface{}) (data.AttributeValue, error) {
	attr := data.NilAttributeValue()
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		attr = data.NewAttributeValueInt(cast.ToInt64(val))
	case float32, float64:
		attr = data.NewAttributeValueDouble(cast.ToFloat64(val))
	case string:
		attr = data.NewAttributeValueString(val)
	case bool:
		attr = data.NewAttributeValueBool(val)
	default:
		return attr, fmt.Errorf("error unsupported value type \"%T\"", value)
	}
	return attr, nil
}
