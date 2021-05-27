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

package goldendataset

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cast"

	"go.opentelemetry.io/collector/internal/data"
	otlpcommon "go.opentelemetry.io/collector/internal/data/protogen/common/v1"
)

func convertMapToAttributeKeyValues(attrsMap map[string]interface{}) []otlpcommon.KeyValue {
	if attrsMap == nil {
		return nil
	}
	attrList := make([]otlpcommon.KeyValue, len(attrsMap))
	index := 0
	for key, value := range attrsMap {
		attrList[index] = constructAttributeKeyValue(key, value)
		index++
	}
	return attrList
}

func constructAttributeKeyValue(key string, value interface{}) otlpcommon.KeyValue {
	var attr otlpcommon.KeyValue
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		attr = otlpcommon.KeyValue{
			Key:   key,
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: cast.ToInt64(val)}},
		}
	case float32, float64:
		attr = otlpcommon.KeyValue{
			Key:   key,
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: cast.ToFloat64(val)}},
		}
	case bool:
		attr = otlpcommon.KeyValue{
			Key:   key,
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BoolValue{BoolValue: cast.ToBool(val)}},
		}
	case *otlpcommon.ArrayValue:
		attr = otlpcommon.KeyValue{
			Key:   key,
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: val}},
		}
	case *otlpcommon.KeyValueList:
		attr = otlpcommon.KeyValue{
			Key:   key,
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: val}},
		}
	default:
		attr = otlpcommon.KeyValue{
			Key:   key,
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: val.(string)}},
		}
	}
	return attr
}

func loadPictOutputFile(fileName string) ([][]string, error) {
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	reader := csv.NewReader(file)
	reader.Comma = '\t'

	return reader.ReadAll()
}

func generateTraceID(random io.Reader) data.TraceID {
	var r [16]byte
	_, err := random.Read(r[:])
	if err != nil {
		panic(err)
	}
	return data.NewTraceID(r)
}

func generateSpanID(random io.Reader) data.SpanID {
	var r [8]byte
	_, err := random.Read(r[:])
	if err != nil {
		panic(err)
	}
	return data.NewSpanID(r)
}
