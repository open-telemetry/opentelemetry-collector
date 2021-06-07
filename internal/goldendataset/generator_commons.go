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

	"go.opentelemetry.io/collector/consumer/pdata"
)

func convertMapToAttributeMap(attrsMap map[string]interface{}) *pdata.AttributeMap {
	attributeMap := pdata.NewAttributeMap()
	if attrsMap == nil {
		return nil
	}
	for key, value := range attrsMap {
		attributeMap.Insert(key, convertToAttributeValue(value))
	}
	return &attributeMap
}

func convertToAttributeValue(value interface{}) pdata.AttributeValue {
	var newValue pdata.AttributeValue
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		newValue = pdata.NewAttributeValueInt(cast.ToInt64(val))
	case float32, float64:
		newValue = pdata.NewAttributeValueDouble(cast.ToFloat64(val))
	case bool:
		newValue = pdata.NewAttributeValueBool(cast.ToBool(val))
	case pdata.AttributeMap:
		newValue = pdata.NewAttributeValueMap()
		val.CopyTo(newValue.MapVal())
	case pdata.AnyValueArray:
		newValue = pdata.NewAttributeValueArray()
		val.CopyTo(newValue.ArrayVal())
	case pdata.AttributeValue:
		newValue = val
	default:
		newValue = pdata.NewAttributeValueString(val.(string))
	}
	return newValue
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

func generatePDataTraceID(random io.Reader) pdata.TraceID {
	var r [16]byte
	_, err := random.Read(r[:])
	if err != nil {
		panic(err)
	}
	return pdata.NewTraceID(r)
}

func generatePDataSpanID(random io.Reader) pdata.SpanID {
	var r [8]byte
	_, err := random.Read(r[:])
	if err != nil {
		panic(err)
	}
	return pdata.NewSpanID(r)
}
