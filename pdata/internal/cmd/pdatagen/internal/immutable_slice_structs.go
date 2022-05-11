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

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

import (
	"os"
	"strings"
)

const immutableSliceTemplate = `// ${structName} represents a []${itemType} slice that cannot be mutated.
// The instance of ${structName} can be assigned to multiple objects since it's immutable.
type ${structName} struct {
	value []${itemType}
}

// New${structName} creates a new ${structName} by copying the provided []${itemType} slice.
func New${structName}(val []${itemType}) ${structName} {
	is := ${structName}{}
	if len(val) != 0 {
		is.value = make([]${itemType}, len(val))
		copy(is.value, val)
	}
	return is
}

// AsRaw returns a copy of the []${itemType} slice.
func (is ${structName}) AsRaw() []${itemType} {
	if len(is.value) == 0 {
		return nil
	}
	val := make([]${itemType}, len(is.value))
	copy(val, is.value)
	return val
}

// Len returns length of the []${itemType} slice value.
func (is ${structName}) Len() int {
	return len(is.value)
}

// At returns an item from particular index.
func (is ${structName}) At(i int) ${itemType} {
	return is.value[i]
}`

const immutableSliceTestTemplate = `func TestNew${structName}(t *testing.T) {
	tests := []struct {
		name string
		orig []${itemType}
		want []${itemType}
	}{
		{
			name: "nil",
			orig: nil,
			want: nil,
		},
		{
			name: "empty",
			orig: []${itemType}{},
			want: nil,
		},
		{
			name: "copy",
			orig: []${itemType}{1, 2, 3},
			want: []${itemType}{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New${structName}(tt.orig)
			assert.Equal(t, tt.want, s.AsRaw())
			assert.Equal(t, len(s.value), s.Len())
			if len(tt.orig) > 0 {
				// verify that orig mutation doesn't have any effect
				tt.orig[0] = ${itemType}(0)
				assert.Equal(t, ${itemType}(1), s.At(0))
			}
		})
	}
}`

const immutableSliceAliasTemplate = `// ${structName} represents a []${itemType} slice that cannot be mutated.
type ${structName} = internal.${structName}

// New${structName} creates a new ${structName} by copying the provided []${itemType} slice.
var New${structName} = internal.New${structName}
`

type immutableSliceStruct struct {
	structName string
	itemType   string
}

func (iss *immutableSliceStruct) getName() string {
	return iss.structName
}

func (iss *immutableSliceStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(immutableSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return iss.structName
		case "itemType":
			return iss.itemType
		default:
			panic(name)
		}
	}))
}

func (iss *immutableSliceStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(immutableSliceTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return iss.structName
		case "itemType":
			return iss.itemType
		default:
			panic(name)
		}
	}))
}

func (iss *immutableSliceStruct) generateTestValueHelpers(*strings.Builder) {}

func (iss *immutableSliceStruct) generateAlias(sb *strings.Builder) {
	sb.WriteString(os.Expand(immutableSliceAliasTemplate, func(name string) string {
		switch name {
		case "structName":
			return iss.structName
		case "itemType":
			return iss.itemType
		default:
			panic(name)
		}
	}))
}

var immutableSliceFile = &File{
	Name:     "immutable_slice",
	IsCommon: true,
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
	},
	structs: []baseStruct{
		immutableByteSliceStruct,
		immutableFloat64SliceStruct,
		immutableUInt64SliceStruct,
	},
}

var immutableByteSliceStruct = &immutableSliceStruct{
	structName: "ImmutableByteSlice",
	itemType:   "byte",
}

var immutableFloat64SliceStruct = &immutableSliceStruct{
	structName: "ImmutableFloat64Slice",
	itemType:   "float64",
}

var immutableUInt64SliceStruct = &immutableSliceStruct{
	structName: "ImmutableUInt64Slice",
	itemType:   "uint64",
}
