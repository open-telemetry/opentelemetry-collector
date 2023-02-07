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
	"bytes"
	"os"
	"strings"
)

const primitiveSliceTemplate = `// ${structName} represents a []${itemType} slice.
// The instance of ${structName} can be assigned to multiple objects since it's immutable.
//
// Must use NewMutable${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	common${structName}
}

type Mutable${structName} struct {
	common${structName}
	preventConversion struct{} // nolint:unused
}

type common${structName} struct {
	orig *[]${itemType}
}

// nolint:unused
func (es ${structName}) asMutable() Mutable${structName} {
	return Mutable${structName}{common${structName}: common${structName}{orig: es.orig}}
}

func (es Mutable${structName}) AsImmutable() ${structName} {
	return ${structName}{common${structName}{orig: es.orig}}
}

func New${structName}FromOrig(orig *[]${itemType}) ${structName} {
	return ${structName}{common${structName}{orig}}
}

func NewMutable${structName}FromOrig(orig *[]${itemType}) Mutable${structName} {
	return Mutable${structName}{common${structName}: common${structName}{orig}}
}

// NewMutable${structName} creates a new empty ${structName}.
func NewMutable${structName}() Mutable${structName} {
	orig := []${itemType}(nil)
	return Mutable${structName}{common${structName}: common${structName}{&orig}}
}

// AsRaw returns a copy of the []${itemType} slice.
func (ms common${structName}) AsRaw() []${itemType} {
	return copy${structName}(nil, *ms.orig)
}

// FromRaw copies raw []${itemType} into the slice ${structName}.
func (ms Mutable${structName}) FromRaw(val []${itemType}) {
	*ms.orig = copy${structName}(*ms.orig, val)
}

// Len returns length of the []${itemType} slice value.
// Equivalent of len(${lowerStructName}).
func (ms common${structName}) Len() int {
	return len(*ms.orig)
}

// At returns an item from particular index.
// Equivalent of ${lowerStructName}[i].
func (ms common${structName}) At(i int) ${itemType} {
	return (*ms.orig)[i]
}

// SetAt sets ${itemType} item at particular index.
// Equivalent of ${lowerStructName}[i] = val
func (ms Mutable${structName}) SetAt(i int, val ${itemType}) {
	(*ms.orig)[i] = val
}

// EnsureCapacity ensures ${structName} has at least the specified capacity.
// 1. If the newCap <= cap, then is no change in capacity.
// 2. If the newCap > cap, then the slice capacity will be expanded to the provided value which will be equivalent of:
//	buf := make([]${itemType}, len(${lowerStructName}), newCap)
//	copy(buf, ${lowerStructName})
//	${lowerStructName} = buf
func (ms Mutable${structName}) EnsureCapacity(newCap int) {
	oldCap := cap(*ms.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]${itemType}, len(*ms.orig), newCap)
	copy(newOrig, *ms.orig)
	*ms.orig = newOrig
}

// Append appends extra elements to ${structName}.
// Equivalent of ${lowerStructName} = append(${lowerStructName}, elms...) 
func (ms Mutable${structName}) Append(elms ...${itemType}) {
	*ms.orig = append(*ms.orig, elms...)
}

// MoveTo moves all elements from the current slice overriding the destination and 
// resetting the current instance to its zero value.
func (ms Mutable${structName}) MoveTo(dest Mutable${structName}) {
	*dest.orig = *ms.orig
	*ms.orig = nil
}

// CopyTo copies all elements from the current slice overriding the destination.
func (ms common${structName}) CopyTo(dest Mutable${structName}) {
	*dest.orig = copy${structName}(*dest.orig, *ms.orig)
}

func copy${structName}(dst, src []${itemType}) []${itemType} {
	dst = dst[:0]
	return append(dst, src...)
}`

const immutableSliceTestTemplate = `func TestNewMutable${structName}(t *testing.T) {
	ms := NewMutable${structName}()
	assert.Equal(t, 0, ms.Len())
	ms.FromRaw([]${itemType}{1, 2, 3})
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, []${itemType}{1, 2, 3}, ms.AsRaw())
	ms.SetAt(1, ${itemType}(5))
	assert.Equal(t, []${itemType}{1, 5, 3}, ms.AsRaw())
	ms.FromRaw([]${itemType}{3})
	assert.Equal(t, 1, ms.Len())
	assert.Equal(t, ${itemType}(3), ms.At(0))
	
	cp := NewMutable${structName}()
	ms.CopyTo(cp)
	ms.SetAt(0, ${itemType}(2))
	assert.Equal(t, ${itemType}(2), ms.At(0))
	assert.Equal(t, ${itemType}(3), cp.At(0))
	ms.CopyTo(cp)
	assert.Equal(t, ${itemType}(2), cp.At(0))
	
	mv := NewMutable${structName}()
	ms.MoveTo(mv)
	assert.Equal(t, 0, ms.Len())
	assert.Equal(t, 1, mv.Len())
	assert.Equal(t, ${itemType}(2), mv.At(0))
	ms.FromRaw([]${itemType}{1, 2, 3})
	ms.MoveTo(mv)
	assert.Equal(t, 3, mv.Len())
	assert.Equal(t, ${itemType}(1), mv.At(0))
}

func Test${structName}Append(t *testing.T) {
	ms := NewMutable${structName}()
	ms.FromRaw([]${itemType}{1, 2, 3})
	ms.Append(4, 5)
	assert.Equal(t, 5, ms.Len())
	assert.Equal(t, ${itemType}(5), ms.At(4))
}

func Test${structName}EnsureCapacity(t *testing.T) {
	ms := NewMutable${structName}()
	ms.EnsureCapacity(4)
	assert.Equal(t, 4, cap(*ms.orig))
	ms.EnsureCapacity(2)
	assert.Equal(t, 4, cap(*ms.orig))
}`

// primitiveSliceStruct generates a struct for a slice of primitive value elements. The structs are always generated
// in a way that they can be used as fields in structs from other packages (using the internal package).
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

func (iss *primitiveSliceStruct) generateStruct(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(primitiveSliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return iss.structName
		case "lowerStructName":
			return strings.ToLower(iss.structName[:1]) + iss.structName[1:]
		case "itemType":
			return iss.itemType
		default:
			panic(name)
		}
	}))
}

func (iss *primitiveSliceStruct) generateTests(sb *bytes.Buffer) {
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

func (iss *primitiveSliceStruct) generateTestValueHelpers(*bytes.Buffer) {}

func (iss *primitiveSliceStruct) generateAliases(sb *bytes.Buffer) {
	sb.WriteString(os.Expand(aliasTemplate, func(name string) string {
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
