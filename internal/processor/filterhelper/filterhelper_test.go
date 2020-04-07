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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

func TestHelper_AttributeValue(t *testing.T) {
	val, err := NewAttributeValue(uint8(123))
	assert.Equal(t, data.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue(uint16(123))
	assert.Equal(t, data.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue(int8(123))
	assert.Equal(t, data.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue(int16(123))
	assert.Equal(t, data.NewAttributeValueInt(123), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue(float32(234.129312))
	assert.Equal(t, data.NewAttributeValueDouble(float64(float32(234.129312))), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue(234.129312)
	assert.Equal(t, data.NewAttributeValueDouble(234.129312), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue(true)
	assert.Equal(t, data.NewAttributeValueBool(true), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue("bob the builder")
	assert.Equal(t, data.NewAttributeValueString("bob the builder"), val)
	assert.NoError(t, err)

	val, err = NewAttributeValue(nil)
	assert.True(t, val.IsNil())
	assert.Error(t, err)

	val, err = NewAttributeValue(t)
	assert.True(t, val.IsNil())
	assert.Error(t, err)
}
