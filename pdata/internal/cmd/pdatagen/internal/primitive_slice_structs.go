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

const primitiveSliceTemplate = `// ${structName} represents a []${itemType} slice.
// The instance of ${structName} can be assigned to multiple objects since it's immutable.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} internal.${structName}

func (ms ${structName}) getOrig() *[]${itemType} {
	return internal.GetOrig${structName}(internal.${structName}(ms))
}

// New${structName} creates a new empty ${structName}.
func New${structName}() ${structName} {
	orig := []${itemType}(nil)
	return ${structName}(internal.New${structName}(&orig))
}

// AsRaw returns a copy of the []${itemType} slice.
func (ms ${structName}) AsRaw() []${itemType} {
	return copy${structName}(*ms.getOrig())
}

// FromRaw copies raw []${itemType} into the slice ${structName}.
func (ms ${structName}) FromRaw(val []${itemType}) {
	*ms.getOrig() = copy${structName}(val)
}

// Len returns length of the []${itemType} slice value.
func (ms ${structName}) Len() int {
	return len(*ms.getOrig())
}

// At returns an item from particular index.
func (ms ${structName}) At(i int) ${itemType} {
	return (*ms.getOrig())[i]
}

// SetAt sets ${itemType} item at particular index.
func (ms ${structName}) SetAt(i int, val ${itemType}) {
	(*ms.getOrig())[i] = val
}

// MoveTo moves ${structName} to another instance.
func (ms ${structName}) MoveTo(dest ${structName}) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = nil
}

// CopyTo copies ${structName} to another instance.
func (ms ${structName}) CopyTo(dest ${structName}) {
	*dest.getOrig() = copy${structName}(*ms.getOrig())
}

func copy${structName}(from []${itemType}) []${itemType} {
	if len(from) == 0 {
		return nil
	}

	to := make([]${itemType}, len(from))
	copy(to, from)
	return to
}`

const immutableSliceTestTemplate = `func TestNew${structName}(t *testing.T) {
	ms := New${structName}()
	assert.Equal(t, 0, ms.Len())
	ms.FromRaw([]${itemType}{1, 2, 3})
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, []${itemType}{1, 2, 3}, ms.AsRaw())
	ms.SetAt(1, ${itemType}(5))
	assert.Equal(t, []${itemType}{1, 5, 3}, ms.AsRaw())
	ms.FromRaw([]${itemType}{3})
	assert.Equal(t, 1, ms.Len())
	assert.Equal(t, ${itemType}(3), ms.At(0))
	
	cp := New${structName}()
	ms.CopyTo(cp)
	ms.SetAt(0, ${itemType}(2))
	assert.Equal(t, ${itemType}(2), ms.At(0))
	assert.Equal(t, ${itemType}(3), cp.At(0))
	ms.CopyTo(cp)
	assert.Equal(t, ${itemType}(2), cp.At(0))
	
	mv := New${structName}()
	ms.MoveTo(mv)
	assert.Equal(t, 0, ms.Len())
	assert.Equal(t, 1, mv.Len())
	assert.Equal(t, ${itemType}(2), mv.At(0))
	ms.FromRaw([]${itemType}{1, 2, 3})
	ms.MoveTo(mv)
	assert.Equal(t, 3, mv.Len())
	assert.Equal(t, ${itemType}(1), mv.At(0))
}`

const primitiveSliceInternalTemplate = `
type ${structName} struct {
	orig *[]${itemType}
}

func GetOrig${structName}(ms ${structName}) *[]${itemType} {
	return ms.orig
}

func New${structName}(orig *[]${itemType}) ${structName} {
	return ${structName}{orig: orig}
}`

type primitiveSliceStruct struct {
	structName  string
	packageName string
	itemType    string
}

func (iss *primitiveSliceStruct) getName() string {
	return iss.structName
}

func (iss *primitiveSliceStruct) getPackageName() string {
	return iss.packageName
}

func (iss *primitiveSliceStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(primitiveSliceTemplate, func(name string) string {
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

func (iss *primitiveSliceStruct) generateTests(sb *strings.Builder) {
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

func (iss *primitiveSliceStruct) generateTestValueHelpers(*strings.Builder) {}

func (iss *primitiveSliceStruct) generateInternal(sb *strings.Builder) {
	sb.WriteString(os.Expand(primitiveSliceInternalTemplate, func(name string) string {
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

var primitiveSliceFile = &File{
	Name:        "primitive_slice",
	PackageName: "pcommon",
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`"go.opentelemetry.io/collector/pdata/internal"`,
	},
	structs: []baseStruct{
		byteSliceStruct,
		float64SliceStruct,
		uInt64SliceStruct,
	},
}

var byteSliceStruct = &primitiveSliceStruct{
	structName:  "ByteSlice",
	packageName: "pcommon",
	itemType:    "byte",
}

var float64SliceStruct = &primitiveSliceStruct{
	structName:  "Float64Slice",
	packageName: "pcommon",
	itemType:    "float64",
}

var uInt64SliceStruct = &primitiveSliceStruct{
	structName:  "UInt64Slice",
	packageName: "pcommon",
	itemType:    "uint64",
}
