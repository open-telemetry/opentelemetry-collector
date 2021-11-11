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

package otlpjson

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestNestedArraySerializesCorrectly(t *testing.T) {
	ava := pdata.NewAttributeValueArray()
	ava.SliceVal().AppendEmpty().SetStringVal("foo")
	ava.SliceVal().AppendEmpty().SetIntVal(42)

	ava2 := pdata.NewAttributeValueArray()
	ava2.SliceVal().AppendEmpty().SetStringVal("bar")
	ava2.SliceVal().AppendEmpty().SetMapVal(pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{
		"baz": pdata.NewAttributeValueBytes([]byte{42}),
	}))
	ava2.CopyTo(ava.SliceVal().AppendEmpty())

	ava.SliceVal().AppendEmpty().SetBoolVal(true)
	ava.SliceVal().AppendEmpty().SetDoubleVal(5.5)

	var buf dataBuffer
	buf.attributeValue(ava2)
	buf.attributeValue(ava)
	got := buf.buf.String()

	assert.Equal(t, 5, ava.SliceVal().Len())
	assert.Equal(t, lines(
		`["bar",{"baz":"Kg=="}]`,
		`["foo",42,["bar",{"baz":"Kg=="}],true,5.5]`,
	), got)
	assert.NoError(t, checkJSON([]byte(got)))
}

func TestNestedMapSerializesCorrectly(t *testing.T) {
	ava := pdata.NewAttributeValueMap()
	av := ava.MapVal()
	av.Insert("foo", pdata.NewAttributeValueString("test"))

	ava2 := pdata.NewAttributeValueMap()
	av2 := ava2.MapVal()
	av2.InsertInt("bar", 13)
	av.Insert("zoo", ava2)

	var buf dataBuffer
	buf.attributeValue(ava2)
	buf.attributeValue(ava)
	got := buf.buf.String()

	assert.Equal(t, 2, ava.MapVal().Len())
	assert.Equal(t, lines(
		`{"bar":13}`,
		`{"foo":"test","zoo":{"bar":13}}`,
	), got)
	assert.NoError(t, checkJSON([]byte(got)))
}

func lines(elems ...string) string {
	return strings.Join(elems, "\n")
}

func checkJSON(data []byte) error {
	var j json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		if err := dec.Decode(&j); err != nil {
			return err
		}
	}
	return nil
}

func compactJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		panic(err)
	}
	return buf.String()
}
