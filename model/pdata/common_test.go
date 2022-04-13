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

package pdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAttributeMapFromMapDeprecated(t *testing.T) {
	rawMap := map[string]Value{
		"k_string": NewValueString("123"),
		"k_int":    NewValueInt(int64(123)),
		"k_double": NewValueDouble(float64(1.23)),
		"k_bool":   NewValueBool(true),
		"k_null":   NewValueEmpty(),
		"k_bytes":  NewValueBytes([]byte{1, 2, 3}),
	}

	am := NewAttributeMapFromMap(rawMap)
	assert.Equal(t, 6, am.Len())

	v, ok := am.Get("k_string")
	assert.True(t, ok)
	assert.Equal(t, "123", v.StringVal())

	v, ok = am.Get("k_int")
	assert.True(t, ok)
	assert.Equal(t, int64(123), v.IntVal())

	v, ok = am.Get("k_double")
	assert.True(t, ok)
	assert.Equal(t, float64(1.23), v.DoubleVal())

	v, ok = am.Get("k_bool")
	assert.True(t, ok)
	assert.Equal(t, true, v.BoolVal())

	v, ok = am.Get("k_null")
	assert.True(t, ok)
	assert.Equal(t, ValueTypeEmpty, v.Type())

	v, ok = am.Get("k_bytes")
	assert.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, v.BytesVal())
}
