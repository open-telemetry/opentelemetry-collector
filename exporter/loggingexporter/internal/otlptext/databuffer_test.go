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

package otlptext

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNestedArraySerializesCorrectly(t *testing.T) {
	ava := pcommon.NewValueSlice()
	ava.Slice().AppendEmpty().SetStr("foo")
	ava.Slice().AppendEmpty().SetInt(42)

	ava2 := pcommon.NewValueSlice()
	ava2.Slice().AppendEmpty().SetStr("bar")
	ava2.CopyTo(ava.Slice().AppendEmpty())

	ava.Slice().AppendEmpty().SetBool(true)
	ava.Slice().AppendEmpty().SetDouble(5.5)

	assert.Equal(t, 5, ava.Slice().Len())
	assert.Equal(t, "[foo, 42, [bar], true, 5.5]", attributeValueToString(ava))
}

func TestNestedMapSerializesCorrectly(t *testing.T) {
	ava := pcommon.NewValueMap()
	av := ava.Map()
	av.PutString("foo", "test")

	av.PutEmptyMap("zoo").PutInt("bar", 13)

	expected := `{
     -> foo: STRING(test)
     -> zoo: MAP({"bar":13})
}`

	assert.Equal(t, 2, ava.Map().Len())
	assert.Equal(t, expected, attributeValueToString(ava))
}
