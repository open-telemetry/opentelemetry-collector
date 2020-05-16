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

package goldendataset

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"

	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	"github.com/spf13/cast"
)

func convertMapToAttributeKeyValues(attrsMap map[string]interface{}) []*otlpcommon.AttributeKeyValue {
	if attrsMap == nil {
		return nil
	}
	attrList := make([]*otlpcommon.AttributeKeyValue, len(attrsMap))
	index := 0
	for key, value := range attrsMap {
		attrList[index] = constructAttributeKeyValue(key, value)
		index++
	}
	return attrList
}

func constructAttributeKeyValue(key string, value interface{}) *otlpcommon.AttributeKeyValue {
	var attr otlpcommon.AttributeKeyValue
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		attr = otlpcommon.AttributeKeyValue{
			Key:      key,
			Type:     otlpcommon.AttributeKeyValue_INT,
			IntValue: cast.ToInt64(val),
		}
	case float32, float64:
		attr = otlpcommon.AttributeKeyValue{
			Key:         key,
			Type:        otlpcommon.AttributeKeyValue_DOUBLE,
			DoubleValue: cast.ToFloat64(val),
		}
	case bool:
		attr = otlpcommon.AttributeKeyValue{
			Key:       key,
			Type:      otlpcommon.AttributeKeyValue_BOOL,
			BoolValue: cast.ToBool(val),
		}
	default:
		attr = otlpcommon.AttributeKeyValue{
			Key:         key,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: val.(string),
		}
	}
	return &attr
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

func generateTraceID(random io.Reader) []byte {
	var r [16]byte
	_, err := random.Read(r[:])
	if err != nil {
		panic(err)
	}
	return r[:]
}

func generateSpanID(random io.Reader) []byte {
	var r [8]byte
	_, err := random.Read(r[:])
	if err != nil {
		panic(err)
	}
	return r[:]
}
