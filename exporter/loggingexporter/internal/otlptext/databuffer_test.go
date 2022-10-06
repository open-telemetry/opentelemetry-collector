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
