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

	"go.opentelemetry.io/collector/model/pdata"
)

func TestNestedArraySerializesCorrectly(t *testing.T) {
	ava := pdata.NewAttributeValueArray()
	ava.SliceValue().AppendEmpty().SetStringValue("foo")
	ava.SliceValue().AppendEmpty().SetIntValue(42)

	ava2 := pdata.NewAttributeValueArray()
	ava2.SliceValue().AppendEmpty().SetStringValue("bar")
	ava2.CopyTo(ava.SliceValue().AppendEmpty())

	ava.SliceValue().AppendEmpty().SetBoolValue(true)
	ava.SliceValue().AppendEmpty().SetDoubleValue(5.5)

	assert.Equal(t, 5, ava.SliceValue().Len())
	assert.Equal(t, "[foo, 42, [bar], true, 5.5]", attributeValueToString(ava))
}

func TestNestedArraySerializesCorrectlyDeprecated(t *testing.T) {
	ava := pdata.NewAttributeValueArray()
	ava.SliceVal().AppendEmpty().SetStringVal("foo")
	ava.SliceVal().AppendEmpty().SetIntVal(42)

	ava2 := pdata.NewAttributeValueArray()
	ava2.SliceVal().AppendEmpty().SetStringVal("bar")
	ava2.CopyTo(ava.SliceVal().AppendEmpty())

	ava.SliceVal().AppendEmpty().SetBoolVal(true)
	ava.SliceVal().AppendEmpty().SetDoubleVal(5.5)

	assert.Equal(t, 5, ava.SliceVal().Len())
	assert.Equal(t, "[foo, 42, [bar], true, 5.5]", attributeValueToString(ava))
}

func TestNestedMapSerializesCorrectly(t *testing.T) {
	ava := pdata.NewAttributeValueMap()
	av := ava.MapValue()
	av.Insert("foo", pdata.NewAttributeValueString("test"))

	ava2 := pdata.NewAttributeValueMap()
	av2 := ava2.MapValue()
	av2.InsertInt("bar", 13)
	av.Insert("zoo", ava2)

	expected := `{
     -> foo: STRING(test)
     -> zoo: MAP({"bar":13})
}`

	assert.Equal(t, 2, ava.MapValue().Len())
	assert.Equal(t, expected, attributeValueToString(ava))
}
