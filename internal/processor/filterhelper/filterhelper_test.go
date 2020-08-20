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
