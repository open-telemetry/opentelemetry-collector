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

package processor

import (
	"fmt"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
)

// AttributeValue is used to convert the raw `value` from ActionKeyValue to the supported trace attribute values.
func AttributeValue(value interface{}) (*tracepb.AttributeValue, error) {
	attrib := &tracepb.AttributeValue{}
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		attrib.Value = &tracepb.AttributeValue_IntValue{IntValue: cast.ToInt64(val)}
	case float32, float64:
		attrib.Value = &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(val)}
	case string:
		attrib.Value = &tracepb.AttributeValue_StringValue{
			StringValue: &tracepb.TruncatableString{Value: val},
		}
	case bool:
		attrib.Value = &tracepb.AttributeValue_BoolValue{BoolValue: val}
	default:
		return nil, fmt.Errorf("error unsupported value type \"%T\"", value)
	}
	return attrib, nil
}
