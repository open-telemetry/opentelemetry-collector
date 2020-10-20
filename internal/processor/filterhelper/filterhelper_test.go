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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestHelper_AttributeValue(t *testing.T) {
	val, err := NewAttributeValueRaw(uint8(123))
	assert.Equal(t, pdata.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValueRaw(uint16(123))
	assert.Equal(t, pdata.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValueRaw(int8(123))
	assert.Equal(t, pdata.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValueRaw(int16(123))
	assert.Equal(t, pdata.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValueRaw(float32(234.129312))
	assert.Equal(t, pdata.NewAttributeValueDouble(float64(float32(234.129312))), val)
	assert.NoError(t, err)

	val, err = NewAttributeValueRaw(234.129312)
	assert.Equal(t, pdata.NewAttributeValueDouble(234.129312), val)
	assert.NoError(t, err)

	val, err = NewAttributeValueRaw(true)
	assert.Equal(t, pdata.NewAttributeValueBool(true), val)
	assert.NoError(t, err)

	val, err = NewAttributeValueRaw("bob the builder")
	assert.Equal(t, pdata.NewAttributeValueString("bob the builder"), val)
	assert.NoError(t, err)

	_, err = NewAttributeValueRaw(nil)
	assert.Error(t, err)

	_, err = NewAttributeValueRaw(t)
	assert.Error(t, err)
}

func TestHelper_AttributeValueArray(t *testing.T) {
	testData := []interface{}{"alpha", "beta", "gamma", "delta"}
	val, err := NewAttributeValueRaw(testData)
	assert.Equal(t, val.Type(), pdata.AttributeValueARRAY)
	assert.NotNil(t, val.ArrayVal())
	assert.Equal(t, 4, val.ArrayVal().Len())
	assert.NoError(t, err)
	for i := 0; i < val.ArrayVal().Len(); i++ {
		assert.Equal(t, pdata.NewAttributeValueString(testData[i].(string)), val.ArrayVal().At(i))
	}

	testData = []interface{}{int64(200), int64(203), int64(500)}
	val, err = NewAttributeValueRaw(testData)
	assert.Equal(t, val.Type(), pdata.AttributeValueARRAY)
	assert.NotNil(t, val.ArrayVal())
	assert.Equal(t, 3, val.ArrayVal().Len())
	assert.NoError(t, err)
	for i := 0; i < val.ArrayVal().Len(); i++ {
		assert.Equal(t, pdata.NewAttributeValueInt(testData[i].(int64)), val.ArrayVal().At(i))
	}

	testData = []interface{}{float32(5341.129312), float32(888.102)}
	val, err = NewAttributeValueRaw(testData)
	assert.Equal(t, val.Type(), pdata.AttributeValueARRAY)
	assert.NotNil(t, val.ArrayVal())
	assert.Equal(t, 2, val.ArrayVal().Len())
	assert.NoError(t, err)
	for i := 0; i < val.ArrayVal().Len(); i++ {
		assert.Equal(t, pdata.NewAttributeValueDouble(float64(testData[i].(float32))), val.ArrayVal().At(i))
	}

	testData = []interface{}{true}
	val, err = NewAttributeValueRaw(testData)
	assert.Equal(t, val.Type(), pdata.AttributeValueARRAY)
	assert.NotNil(t, val.ArrayVal())
	assert.Equal(t, 1, val.ArrayVal().Len())
	assert.NoError(t, err)
	for i := 0; i < val.ArrayVal().Len(); i++ {
		assert.Equal(t, pdata.NewAttributeValueBool(testData[i].(bool)), val.ArrayVal().At(i))
	}

	var southWeather = make(map[string]interface{})
	southWeather["NY"] = 15.9
	southWeather["VT"] = 13.1
	southWeather["NH"] = 18.0
	val, err = NewAttributeValueRaw(southWeather)
	assert.Equal(t, pdata.AttributeValue{}, val)
	assert.Error(t, err)

	var northWeather = make(map[string]interface{})
	northWeather["AR"] = 45.7
	northWeather["TX"] = 43.3
	northWeather["MS"] = 41.1

	val, err = NewAttributeValueRaw([]interface{}{southWeather, northWeather})
	assert.Equal(t, pdata.AttributeValue{}, val)
	assert.Error(t, err)
}
