// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	"math/rand"
	"testing"

	otlp "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	"github.com/stretchr/testify/assert"
)

func TestAttributeValue(t *testing.T) {
	v := NewAttributeValueString("abc")
	assert.EqualValues(t, otlp.AttributeKeyValue_STRING, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v = NewAttributeValueInt(123)
	assert.EqualValues(t, otlp.AttributeKeyValue_INT, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v = NewAttributeValueDouble(3.4)
	assert.EqualValues(t, otlp.AttributeKeyValue_DOUBLE, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v = NewAttributeValueBool(true)
	assert.EqualValues(t, otlp.AttributeKeyValue_BOOL, v.Type())
	assert.EqualValues(t, true, v.BoolVal())

	v.SetString("abc")
	assert.EqualValues(t, otlp.AttributeKeyValue_STRING, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v.SetInt(123)
	assert.EqualValues(t, otlp.AttributeKeyValue_INT, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v.SetDouble(3.4)
	assert.EqualValues(t, otlp.AttributeKeyValue_DOUBLE, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v.SetBool(true)
	assert.EqualValues(t, otlp.AttributeKeyValue_BOOL, v.Type())
	assert.EqualValues(t, true, v.BoolVal())
}

func TestNewAttributeValueSlice(t *testing.T) {
	events := NewAttributeValueSlice(0)
	assert.EqualValues(t, 0, len(events))

	n := rand.Intn(10)
	events = NewAttributeValueSlice(n)
	assert.EqualValues(t, n, len(events))
	for event := range events {
		assert.NotNil(t, event)
	}
}
