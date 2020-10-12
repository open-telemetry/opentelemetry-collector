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

package filterhelper

import (
	"fmt"

	"github.com/spf13/cast"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// NewAttributeValueRaw is used to convert the raw `value` from ActionKeyValue to the supported trace attribute values.
// If error different than nil the return value is invalid. Calling any functions on the invalid value will cause a panic.
func NewAttributeValueRaw(value interface{}) (pdata.AttributeValue, error) {
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return pdata.NewAttributeValueInt(cast.ToInt64(val)), nil
	case float32, float64:
		return pdata.NewAttributeValueDouble(cast.ToFloat64(val)), nil
	case string:
		return pdata.NewAttributeValueString(val), nil
	case bool:
		return pdata.NewAttributeValueBool(val), nil
	case []interface{}:
		return fromArray(val), nil
	case map[string]interface{}:
		return fromMap(val), nil
	default:
		return pdata.AttributeValue{}, fmt.Errorf("error unsupported value type \"%T\"", value)
	}
}

func fromArray(val []interface{}) pdata.AttributeValue {
	arr := pdata.NewAnyValueArray()
	for _, v := range val {
		arr.Append(fromVal(v))
	}
	av := pdata.NewAttributeValueArray()
	av.SetArrayVal(arr)
	return av
}

func fromMap(v map[string]interface{}) pdata.AttributeValue {
	m := pdata.NewAttributeMap()
	for k, v := range v {
		m.Insert(k, fromVal(v))
	}
	m.Sort()
	av := pdata.NewAttributeValueMap()
	av.SetMapVal(m)
	return av
}

func fromVal(val interface{}) pdata.AttributeValue {
	switch v := val.(type) {
	case string:
		return pdata.NewAttributeValueString(v)
	case int:
		return pdata.NewAttributeValueInt(int64(v))
	case float64:
		return pdata.NewAttributeValueDouble(v)
	case []interface{}:
		return fromArray(v)
	case map[string]interface{}:
		return fromMap(v)
	}
	panic("data type is not supported in fromVal()")
}
