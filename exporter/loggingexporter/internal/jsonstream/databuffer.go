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
	"encoding/base64"
	"strconv"

	"go.opentelemetry.io/collector/model/pdata"
)

type dataBuffer struct {
	buf bytes.Buffer

	depth int
	comma bool
}

func (b *dataBuffer) tokenComma() {
	b.buf.WriteByte(',')
}

func (b *dataBuffer) tokenSemicolon() {
	b.buf.WriteByte(':')
}

func (b *dataBuffer) tokenNull() {
	b.buf.WriteString("null")
}

func (b *dataBuffer) tokenString(s string) {
	b.buf.WriteString(strconv.Quote(s))
}

func (b *dataBuffer) tokenUint32(n uint32) {
	b.tokenUint64(uint64(n))
}

func (b *dataBuffer) tokenUint64(n uint64) {
	b.buf.WriteString(strconv.FormatUint(n, 10))
}

func (b *dataBuffer) tokenInt32(n int32) {
	b.tokenInt64(int64(n))
}

func (b *dataBuffer) tokenInt64(n int64) {
	b.buf.WriteString(strconv.FormatInt(n, 10))
}

func (b *dataBuffer) tokenFloat64(f float64) {
	b.buf.WriteString(strconv.FormatFloat(f, 'f', -1, 64))
}

func (b *dataBuffer) tokenBool(v bool) {
	b.buf.WriteString(strconv.FormatBool(v))
}

func (b *dataBuffer) objectOrArray(left, right byte, f func()) {
	b.element(func() {
		b.depth++
		b.buf.WriteByte(left)
		f()
		b.buf.WriteByte(right)
		b.depth--
	})
}

func (b *dataBuffer) object(f func()) {
	b.objectOrArray('{', '}', f)
}

func (b *dataBuffer) array(f func()) {
	b.objectOrArray('[', ']', f)
}

func (b *dataBuffer) element(f func()) {
	if b.comma {
		if b.depth != 0 {
			b.tokenComma()
		} else {
			b.buf.WriteByte('\n')
		}
	}
	b.comma = false
	f()
	b.comma = true
}

func (b *dataBuffer) elementUint64(v uint64) {
	b.element(func() {
		b.tokenUint64(v)
	})
}

func (b *dataBuffer) elementFloat64(v float64) {
	b.element(func() {
		b.tokenFloat64(v)
	})
}

func (b *dataBuffer) field(name string, f func()) {
	b.element(func() {
		b.tokenString(name)
		b.tokenSemicolon()
		f()
	})
}

func (b *dataBuffer) attributeValue(v pdata.AttributeValue) {
	b.element(func() {
		switch v.Type() {
		case pdata.AttributeValueTypeEmpty:
			b.tokenNull()
		case pdata.AttributeValueTypeString:
			b.tokenString(v.StringVal())
		case pdata.AttributeValueTypeBool:
			b.tokenBool(v.BoolVal())
		case pdata.AttributeValueTypeDouble:
			b.tokenFloat64(v.DoubleVal())
		case pdata.AttributeValueTypeInt:
			b.tokenInt64(v.IntVal())
		case pdata.AttributeValueTypeMap:
			b.attributeMap(v.MapVal())
		case pdata.AttributeValueTypeBytes:
			b.tokenString(base64.StdEncoding.EncodeToString(v.BytesVal()))
		case pdata.AttributeValueTypeArray:
			b.attributeValueSlice(v.SliceVal())
		default:
			b.tokenString(v.AsString())
		}
	})
}

func (b *dataBuffer) attributeValueSlice(s pdata.AttributeValueSlice) {
	b.array(func() {
		for i := 0; i < s.Len(); i++ {
			b.attributeValue(s.At(i))
		}
	})
}

func (b *dataBuffer) attributeMap(m pdata.AttributeMap) {
	b.object(func() {
		m.Range(func(k string, v pdata.AttributeValue) bool {
			b.fieldAttr(k, v)
			return true
		})
	})
}

func (b *dataBuffer) fieldString(name, value string) {
	b.field(name, func() {
		b.tokenString(value)
	})
}

func (b *dataBuffer) fieldUint32(name string, value uint32) {
	b.field(name, func() {
		b.tokenUint32(value)
	})
}

func (b *dataBuffer) fieldUint64(name string, value uint64) {
	b.field(name, func() {
		b.tokenUint64(value)
	})
}

func (b *dataBuffer) fieldInt32(name string, value int32) {
	b.field(name, func() {
		b.tokenInt32(value)
	})
}

func (b *dataBuffer) fieldInt64(name string, value int64) {
	b.field(name, func() {
		b.tokenInt64(value)
	})
}

func (b *dataBuffer) fieldFloat64(name string, value float64) {
	b.field(name, func() {
		b.tokenFloat64(value)
	})
}

func (b *dataBuffer) fieldBool(name string, value bool) {
	b.field(name, func() {
		b.tokenBool(value)
	})
}

func (b *dataBuffer) fieldTime(name string, value pdata.Timestamp) {
	b.field(name, func() {
		b.tokenString(value.String())
	})
}

func (b *dataBuffer) fieldArray(name string, f func()) {
	b.field(name, func() {
		b.array(f)
	})
}

func (b *dataBuffer) fieldObject(name string, f func()) {
	b.field(name, func() {
		b.object(f)
	})
}

func (b *dataBuffer) fieldAttr(name string, v pdata.AttributeValue) {
	b.field(name, func() {
		b.attributeValue(v)
	})
}

func (b *dataBuffer) fieldAttrs(name string, m pdata.AttributeMap) {
	b.field(name, func() {
		b.attributeMap(m)
	})
}

func (b *dataBuffer) instrumentationLibrary(lib pdata.InstrumentationLibrary) {
	b.fieldObject("instrumentationLibrary", func() {
		b.fieldString("name", lib.Name())
		b.fieldString("version", lib.Version())
	})
}

func (b *dataBuffer) resource(typ string, attrs pdata.AttributeMap) {
	b.fieldObject("resource", func() {
		b.fieldString("type", typ)
		b.fieldAttrs("labels", attrs)
	})
}
