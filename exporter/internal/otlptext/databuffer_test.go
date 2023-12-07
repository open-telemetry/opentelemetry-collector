// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	ava.Slice().AppendEmpty().SetEmptySlice().AppendEmpty().SetStr("bar")
	ava.Slice().AppendEmpty().SetBool(true)
	ava.Slice().AppendEmpty().SetDouble(5.5)

	assert.Equal(t, `Slice(["foo",42,["bar"],true,5.5])`, valueToString(ava))
}

func TestNestedMapSerializesCorrectly(t *testing.T) {
	ava := pcommon.NewValueMap()
	av := ava.Map()
	av.PutStr("foo", "test")
	av.PutEmptyMap("zoo").PutInt("bar", 13)

	assert.Equal(t, `Map({"foo":"test","zoo":{"bar":13}})`, valueToString(ava))
}
