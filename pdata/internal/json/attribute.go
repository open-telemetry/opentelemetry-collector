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
	"encoding/base64"
	"fmt"

	jsoniter "github.com/json-iterator/go"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

// ReadAttribute Unmarshal JSON data and return otlpcommon.KeyValue
func ReadAttribute(iter *jsoniter.Iterator) otlpcommon.KeyValue {
	kv := otlpcommon.KeyValue{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "key":
			kv.Key = iter.ReadString()
		case "value":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				kv.Value = ReadAnyValue(iter, f)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return kv
}

// ReadAnyValue Unmarshal JSON data and return otlpcommon.AnyValue
func ReadAnyValue(iter *jsoniter.Iterator, f string) otlpcommon.AnyValue {
	switch f {
	case "stringValue", "string_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_StringValue{
				StringValue: iter.ReadString(),
			},
		}
	case "boolValue", "bool_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_BoolValue{
				BoolValue: iter.ReadBool(),
			},
		}
	case "intValue", "int_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_IntValue{
				IntValue: ReadInt64(iter),
			},
		}
	case "doubleValue", "double_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_DoubleValue{
				DoubleValue: ReadFloat64(iter),
			},
		}
	case "bytesValue", "bytes_value":
		v, err := base64.StdEncoding.DecodeString(iter.ReadString())
		if err != nil {
			iter.ReportError("bytesValue", fmt.Sprintf("base64 decode:%v", err))
			return otlpcommon.AnyValue{}
		}
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_BytesValue{
				BytesValue: v,
			},
		}
	case "arrayValue", "array_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_ArrayValue{
				ArrayValue: readArray(iter),
			},
		}
	case "kvlistValue", "kvlist_value":
		return otlpcommon.AnyValue{
			Value: &otlpcommon.AnyValue_KvlistValue{
				KvlistValue: readKvlistValue(iter),
			},
		}
	default:
		return otlpcommon.AnyValue{}
	}
}

func readArray(iter *jsoniter.Iterator) *otlpcommon.ArrayValue {
	v := &otlpcommon.ArrayValue{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "values":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
					v.Values = append(v.Values, ReadAnyValue(iter, f))
					return true
				})
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return v
}

func readKvlistValue(iter *jsoniter.Iterator) *otlpcommon.KeyValueList {
	v := &otlpcommon.KeyValueList{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "values":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				v.Values = append(v.Values, ReadAttribute(iter))
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return v
}
