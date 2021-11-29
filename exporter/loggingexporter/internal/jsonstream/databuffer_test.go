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

package jsonstream

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
	m := pdata.NewAttributeValueMap()
	pdata.NewAttributeMapFromMap(map[string]pdata.AttributeValue{
		"baz": pdata.NewAttributeValueBytes([]byte{42}),
	}).CopyTo(m.MapVal())
	m.CopyTo(ava2.SliceVal().AppendEmpty())
	ava2.CopyTo(ava.SliceVal().AppendEmpty())

	ava.SliceVal().AppendEmpty().SetBoolVal(true)
	ava.SliceVal().AppendEmpty().SetDoubleVal(5.5)

	var buf dataBuffer
	buf.attributeValue(ava2)
	buf.attributeValue(ava)
	got := buf.buf.String()

	assert.Equal(t, 5, ava.SliceVal().Len())
	testJSON(t, lines(
		`["bar",{"baz":"Kg=="}]`,
		`["foo",42,["bar",{"baz":"Kg=="}],true,5.5]`,
	), got)
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
	testJSON(t, lines(
		`{"bar":13}`,
		`{"foo":"test","zoo":{"bar":13}}`,
	), got)
}

func lines(elems ...string) string {
	return strings.Join(elems, "\n")
}

func compactJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		panic(err)
	}
	return buf.String()
}

func splitStream(data string) ([]json.RawMessage, error) {
	var elems []json.RawMessage

	dec := json.NewDecoder(strings.NewReader(data))
	for dec.More() {
		var j json.RawMessage
		if err := dec.Decode(&j); err != nil {
			return nil, err
		}
		elems = append(elems, j)
	}

	return elems, nil
}

func streamToArray(data string) (string, error) {
	elems, err := splitStream(data)
	if err != nil {
		return "", err
	}

	raw, err := json.Marshal(elems)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "\t"); err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil
}

func testJSON(t *testing.T, expect, actual string) {
	t.Helper()

	actualArray, err := streamToArray(actual)
	if !assert.NoError(t, err) {
		return
	}
	if actual == expect {
		return
	}

	expectArray, err := streamToArray(expect)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, expectArray, actualArray)
}
